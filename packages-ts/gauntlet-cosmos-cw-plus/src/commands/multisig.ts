import { Result } from '@chainlink/gauntlet-core'
import { time, logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { CosmosCommand, TransactionResponse } from '@chainlink/gauntlet-cosmos'
import { AccAddress } from '@chainlink/gauntlet-cosmos'
import { isDeepEqual } from '../lib/utils'
import { fetchProposalState, makeInspectionMessage } from './inspect'
import { Vote, Cw3WasmMsg, Action, State, Cw3BankMsg, Expiration } from '../lib/types'

import { toUtf8 } from '@cosmjs/encoding'
import { EncodeObject } from '@cosmjs/proto-signing'
import { MsgExecuteContractEncodeObject } from '@cosmjs/cosmwasm-stargate'
import { MsgExecuteContract } from 'cosmjs-types/cosmwasm/wasm/v1/tx'
import { MsgSend } from 'cosmjs-types/cosmos/bank/v1beta1/tx'

export const DEFAULT_VOTING_PERIOD_IN_SECS = 24 * 60 * 60

type ProposalAction = (signer: AccAddress, proposalId: number, message: EncodeObject[]) => Promise<object>

export const wrapCommand = (command) => {
  return class Multisig extends CosmosCommand {
    command: CosmosCommand
    multisig: AccAddress

    static id = `${command.id}:multisig`

    constructor(flags, args) {
      super(flags, args)
    }

    buildCommand = async (flags, args): Promise<CosmosCommand> => {
      this.multisig = process.env.CW3_FLEX_MULTISIG as AccAddress
      if (!AccAddress.validate(this.multisig)) throw new Error(`Invalid Multisig wallet address`)
      if (!AccAddress.validate(process.env.CW4_GROUP)) throw new Error(`Invalid Multisig group address`)

      const c = new command(flags, args) as CosmosCommand
      await c.invokeMiddlewares(c, c.middlewares)
      this.command = c.buildCommand ? await c.buildCommand(flags, args) : c
      return this.command
    }

    makeRawTransaction = async (signer: AccAddress, state?: State) => {
      const messages = await this.command.makeRawTransaction(this.multisig)
      await this.command.simulate(this.multisig, messages)
      logger.info(`Command simulation successful.`)

      const operations = {
        [Action.CREATE]: this.makeProposeTransaction,
        [Action.APPROVE]: this.makeAcceptTransaction,
        [Action.EXECUTE]: this.makeExecuteTransaction,
        [Action.NONE]: () => {
          throw new Error('No action needed')
        },
      }

      if (state.proposal.nextAction !== Action.CREATE) {
        this.require(
          this.isSameProposal(
            state.proposal.data,
            messages.map((element) => this.toMsg(element)),
          ),
          'The transaction generated is different from the proposal provided',
        )
      }

      const proposal_id = Number(this.flags.proposal || this.flags.multisigProposal) // alias requested by eng ops
      let input = operations[state.proposal.nextAction](signer, Number(proposal_id), messages)

      const msg: MsgExecuteContractEncodeObject = {
        typeUrl: '/cosmwasm.wasm.v1.MsgExecuteContract',
        value: MsgExecuteContract.fromPartial({
          sender: signer,
          contract: this.multisig,
          msg: toUtf8(JSON.stringify(input)),
          funds: [],
        }),
      }
      return [msg]
    }

    isSameProposal = (proposalMsgs: (Cw3WasmMsg | Cw3BankMsg)[], generatedMsgs: (Cw3WasmMsg | Cw3BankMsg)[]) => {
      return isDeepEqual(proposalMsgs, generatedMsgs)
    }

    toMsg = (message: EncodeObject): Cw3BankMsg | Cw3WasmMsg => {
      if ('amount' in message.value) return this.toBankMsg(message.value as MsgSend)
      if ('contract' in message.value) return this.toWasmMsg(message.value as MsgExecuteContract)
    }

    toBankMsg = (message: MsgSend): Cw3BankMsg => {
      return {
        bank: {
          send: {
            amount: message.amount,
            to_address: message.toAddress,
          },
        },
      }
    }

    toWasmMsg = (message: MsgExecuteContract): Cw3WasmMsg => {
      return {
        wasm: {
          execute: {
            contract_addr: message.contract,
            funds: message.funds,
            msg: Buffer.from(JSON.stringify(message.msg)).toString('base64'),
          },
        },
      }
    }

    async parseVotingPeriod(votingPeriod: string): Promise<number> {
      const state = await this.fetchState()
      const n = Number.parseInt(votingPeriod)
      if (isNaN(n) || n < 0 || n > state.multisig.maxVotingPeriod) {
        throw new Error(
          `votingPeriod=${votingPeriod}: must be a valid duration in seconds, ` +
            `(range: [0-${state.multisig.maxVotingPeriod}], default: ${DEFAULT_VOTING_PERIOD_IN_SECS})`,
        )
      }
      return n
    }

    makeProposeTransaction: ProposalAction = async (signer, _, message) => {
      // default voting period of 24 hours, if unspecified
      const votingPeriod: number = this.flags.votingPeriod
        ? await this.parseVotingPeriod(this.flags.votingPeriod)
        : DEFAULT_VOTING_PERIOD_IN_SECS

      const msExpiration: number = Date.now() + votingPeriod * 1000 // milliseconds since 1970
      logger.info(`Generating data for creating new multisig proposal (expires at ${new Date(msExpiration)})`)
      // Expiration.at_time is a string representation of nanoseconds since 1970
      const expiration: Expiration = { at_time: (msExpiration * time.Millisecond).toString() }

      const proposeInput = {
        propose: {
          description: command.id,
          msgs: message.map((element) => this.toMsg(element)),
          title: command.id,
          latest: expiration,
        },
      }

      return proposeInput
    }

    makeAcceptTransaction: ProposalAction = async (signer, proposalId, _) => {
      logger.info(`Generating data for approving proposal ${proposalId}`)
      const approvalInput = {
        vote: {
          vote: Vote.YES,
          proposal_id: proposalId,
        },
      }
      return approvalInput
    }

    makeExecuteTransaction: ProposalAction = async (signer, proposalId, _) => {
      logger.info(`Generating data for executing multisig proposal ${proposalId}`)
      const executeInput = {
        execute: {
          proposal_id: proposalId,
        },
      }
      return executeInput
    }

    fetchState = async (proposalId?: number): Promise<State> => {
      return fetchProposalState(this.provider)(this.multisig, proposalId)
    }

    printPostInstructions = async (proposalId: number) => {
      const state = await this.fetchState(proposalId)
      if (!state.proposal.id) {
        logger.error(`Multisig proposal ${proposalId} not found`)
        return
      }
      const approvalsLeft = state.multisig.threshold - state.proposal.approvers.length
      const messages = {
        passed: `The multisig proposal reached the threshold and can be executed. Run the same command with the flag --multisigProposal=${proposalId}`,
        open: `The multisig proposal needs ${approvalsLeft} more approvals. Run the same command with the flag --multisigProposal=${proposalId}`,
        pending: `The multisig proposal needs ${approvalsLeft} more approvals. Run the same command with the flag --multisigProposal=${proposalId}`,
        rejected: `The multisig proposal has been rejected. No actions available`,
        executed: `The multisig proposal has been executed. No more actions needed`,
      }
      logger.line()
      logger.info(`${messages[state.proposal.currentStatus]}`)
      logger.line()
    }

    execute = async () => {
      // TODO: Gauntlet core should initialize commands using `buildCommand` instead of new Command
      await this.buildCommand(this.flags, this.args)

      let proposalId = Number(this.flags.proposal || this.flags.multisigProposal) // alias requested by eng ops
      const state = await this.fetchState(proposalId)
      logger.info(makeInspectionMessage(state))

      if (state.proposal.nextAction === Action.NONE) {
        await this.printPostInstructions(proposalId)
        return
      }
      const rawTx = await this.makeRawTransaction(this.signer.address, state)

      const actionMessage = {
        [Action.CREATE]: 'CREATING',
        [Action.APPROVE]: 'APPROVING',
        [Action.EXECUTE]: 'EXECUTING',
      }

      if (this.flags.execute) {
        await this.command.beforeExecute(this.multisig)

        await prompt(`Continue ${actionMessage[state.proposal.nextAction]} multisig proposal?`)
        const tx = await this.signAndSend(rawTx)
        let response: Result<TransactionResponse> = {
          responses: [
            {
              tx,
              contract: this.multisig,
            },
          ],
          data: {
            proposalId,
          },
        }

        if (state.proposal.nextAction === Action.CREATE) {
          // const proposalFromEvent = tx.events[0].wasm.proposal_id[0] TODO
          const proposalFromEvent = 'a'
          logger.success(`New proposal created with multisig proposal ID: ${proposalFromEvent}`)
          proposalId = Number(proposalFromEvent)
        }

        if (state.proposal.nextAction === Action.EXECUTE && this.command.afterExecute) {
          const data = await this.command.afterExecute(response)
          response = { ...response, data: { ...data } }
        }

        logger.success(`TX finished at ${tx.hash}`)
        await this.printPostInstructions(proposalId)

        return response
      }

      // TODO: Test raw message
      const msgData = Buffer.from(JSON.stringify(rawTx[0].value)).toString('base64')
      logger.line()
      logger.success(`Message generated succesfully for ${actionMessage[state.proposal.nextAction]} multisig proposal`)
      logger.log()
      logger.log(msgData)
      logger.log()
      logger.line()

      return {
        responses: [
          {
            tx: {},
            contract: this.multisig,
          },
        ],
        data: {
          proposalId,
          message: msgData,
        },
      } as Result<TransactionResponse>
    }
  }
}
