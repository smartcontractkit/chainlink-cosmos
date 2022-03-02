import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { Result } from '@chainlink/gauntlet-core'
import { BN, logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { CATEGORIES } from '../../../lib/constants'
import { io } from '@chainlink/gauntlet-core/dist/utils'
import { assert } from 'console'

interface LinkContractInput {
  transfer: {
    recipient: string
    amount: number
  }
}

export default class TransferLink extends TerraCommand {
  static description = 'Transfer LINK'
  static examples = [
    `yarn gauntlet token:transfer --network=bombay-testnet --to=[RECEIVER] --amount=[AMOUNT_IN_TOKEN_UNITS]`,
    `yarn gauntlet token:transfer --network=bombay-testnet --to=[RECEIVER] --amount=[AMOUNT_IN_TOKEN_UNITS] --link=[TOKEN_ADDRESS] --decimals=[TOKEN_DECIMALS]`,
  ]

  static id = 'token:transfer'
  static category = CATEGORIES.LINK
  static flags = {
    decimals: { description: 'Digits after decimal point to include in amount' },
    amount: { description: 'Amount of LINK to transfer, as a decimal number' },
    to: { description: 'Address of destination wallet' },
    batch: {
      description:
        'Name of JSON file with a list of transfers to execute at once:  [ { "to" : ADDRESS, "amount": AMOUNT }, ... ]',
    },
  }

  constructor(flags, args: string[]) {
    super(flags, args)
  }

  makeRawTransaction = async () => {
    throw new Error('Transfer LINK command: makeRawTransaction method not implemented')
  }

  validateBatchInputs = (jsonFile, inputs) => {
    if (!Array.isArray(inputs)) {
      throw Error(`Invalid contents of ${jsonFile}, must be a list: ${inputs}`)
    } else if (inputs.length == 0) {
      throw Error(`Read empty list from ${jsonFile}, aborting.`)
    }
  }

  makeContractInput = (flags): LinkContractInput => {
    const decimals = this.flags.decimals || 18
    let amount
    // Do we really want to round this to nearest integer, ignoring everything after decimal?
    //const makeAmount = (amount) => new BN(amount).mul(new BN(10).pow(new BN(decimals)))
    try {
      amount = Number.parseFloat(flags.amount) * Math.pow(10, decimals)
    } catch {
      throw Error(`Failed to parse float ${flags.amount}`)
    }
    return {
      transfer: {
        recipient: flags.to,
        amount: amount.toString(),
      },
    } as LinkContractInput
  }

  confirmBatchSend = async (jsonInputs, inputs) => {
    assert(jsonInputs.length == inputs.length, 'Length mismatch')
    for (let i = 0; i < inputs.length; i++) {
      logger.info(
        `About to Send ${jsonInputs[i].amount} LINK (${inputs[i].transfer.amount}) to ${inputs[i].transfer.recipient}.`,
      )
    }
    await prompt(`Send all?`)
  }

  execute = async () => {
    const link = this.flags.link || process.env.LINK
    const batchFile: string = this.flags.batch
    let tx

    if (this.flags.batch) {
      const jsonInputs = io.readJSON(batchFile)
      this.validateBatchInputs(this.flags.batch, jsonInputs)
      const inputs: LinkContractInput[] = jsonInputs.map(this.makeContractInput)
      await this.confirmBatchSend(jsonInputs, inputs)
      tx = await this.callBatch(this.flags.link, inputs)
      logger.success(`LINK transfers were successful (txhash: ${tx.hash})`)
    } else {
      const contractInput = this.makeContractInput(this.flags)

      await prompt(
        `Sending ${this.flags.amount} LINK (${contractInput.transfer.amount}) to ${this.flags.to}. Continue?`,
      )
      tx = await this.call(link, contractInput)
      logger.success(`LINK transferred successfully to ${this.flags.to} (txhash: ${tx.hash})`)
    }
    return {
      responses: [
        {
          tx,
          contract: link,
        },
      ],
    } as Result<TransactionResponse>
  }
}
