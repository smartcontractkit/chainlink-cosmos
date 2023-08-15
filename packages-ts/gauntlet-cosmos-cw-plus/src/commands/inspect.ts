import { CosmosCommand, TransactionResponse, logger, Client } from '@chainlink/gauntlet-cosmos'
import { Result } from '@chainlink/gauntlet-core'
import { Action, State, Vote } from '../lib/types'
import { AccAddress } from '@chainlink/gauntlet-cosmos'

export default class Inspect extends CosmosCommand {
  static id = 'cw3_flex_multisig:inspect'
  static examples = [
    'cw3_flex_multisig:inspect --network=<NETWORK> --multisigProposal=<PROPOSAL_ID> <CW3_FLEX_MULTISIG_ADDRESS>',
  ]

  constructor(flags, args: string[]) {
    super(flags, args)
  }

  makeRawTransaction = async () => {
    throw new Error('Query method does not have any tx')
  }

  fetchState = async (multisig: string, proposalId?: number): Promise<State> => {
    return fetchProposalState(this.provider)(multisig, proposalId)
  }

  execute = async () => {
    const msig = this.args[0] || process.env.CW3_FLEX_MULTISIG
    const proposalId = Number(this.flags.proposal || this.flags.multisigProposal) // alias requested by eng ops
    const state = await this.fetchState(msig, proposalId)

    logger.info(makeInspectionMessage(state))
    return {} as Result<TransactionResponse>
  }
}

export const fetchProposalState =
  (provider: Client) =>
  async (multisig: string, proposalId?: number): Promise<State> => {
    const _queryMultisig = (params) => (): Promise<any> => provider.queryContractSmart(multisig, params)
    const _queryContractInfo = (): Promise<any> => provider.getContract(multisig)

    const multisigQueries = [
      _queryContractInfo,
      _queryMultisig({
        list_voters: {},
      }),
      _queryMultisig({
        threshold: {},
      }),
    ]
    const proposalQueries = [
      _queryMultisig({
        proposal: {
          proposal_id: proposalId,
        },
      }),
      _queryMultisig({
        list_votes: {
          proposal_id: proposalId,
        },
      }),
    ]
    const queries = !!proposalId ? multisigQueries.concat(proposalQueries) : multisigQueries

    const [contractInfo, groupState, thresholdState, proposalState, votes] = await Promise.all(queries.map((q) => q()))

    const admin = (await provider.queryContractSmart(contractInfo.init_msg.group_addr, { admin: {} })) as any
    const multisigState = {
      address: multisig,
      threshold: thresholdState.absolute_count.weight,
      owners: groupState.voters.map((m) => m.addr),
      maxVotingPeriod: contractInfo.init_msg.max_voting_period.time,
      admin: admin?.admin as AccAddress,
      groupAddress: contractInfo.init_msg.group_addr,
    }
    if (!proposalId) {
      return {
        multisig: multisigState,
        proposal: {
          nextAction: Action.CREATE,
          approvers: [],
        },
      }
    }
    const toNextAction = {
      passed: Action.EXECUTE,
      open: Action.APPROVE,
      pending: Action.APPROVE,
      rejected: Action.NONE,
      executed: Action.NONE,
    }
    return {
      multisig: multisigState,
      proposal: {
        id: proposalId,
        nextAction: toNextAction[proposalState.status],
        currentStatus: proposalState.status,
        data: proposalState.msgs,
        approvers: votes.votes.filter((v) => v.vote === Vote.YES).map((v) => v.voter),
        expiresAt: proposalState.expires.at_time ? new Date(proposalState.expires.at_time / 1e6) : null,
      },
    }
  }

export const makeInspectionMessage = (state: State): string => {
  const newline = `\n`
  const indent = '  '.repeat(2)
  const ownersList = state.multisig.owners.map((o) => `\n${indent.repeat(2)} - ${logger.styleAddress(o)}`).join('')
  const multisigMessage = `Multisig State (${state.multisig.address.toString()}):
    - Threshold: ${state.multisig.threshold}
    - Total Owners: ${state.multisig.owners.length}
    - Owners List: ${ownersList}
    - Admin: ${state.multisig.admin.toString()}
    - Group Contract: ${state.multisig.groupAddress.toString()}`

  let proposalMessage = `Proposal State:
    - Next Action: ${state.proposal.nextAction.toUpperCase()}`

  if (!state.proposal.id) return multisigMessage.concat(newline)

  const approversList = state.proposal.approvers
    .map((a) => `\n${indent.repeat(2)} - ${logger.styleAddress(a)}`)
    .join('')
  proposalMessage = proposalMessage.concat(`
    - Multisig Proposal ID: ${state.proposal.id}
    - Total Approvers: ${state.proposal.approvers.length}
    - Approvers List: ${approversList}
    `)

  if (state.proposal.expiresAt) {
    const expiration = `- Approvals expire at ${state.proposal.expiresAt}`
    proposalMessage = proposalMessage.concat(expiration)
  }

  return multisigMessage.concat(newline).concat(proposalMessage).concat(newline)
}
