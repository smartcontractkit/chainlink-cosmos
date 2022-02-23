import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { Result } from '@chainlink/gauntlet-core'
import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { Action, State, Vote } from '../lib/types'

export default class Inspect extends TerraCommand {
  static id = 'cw3_flex_multisig:inspect'

  constructor(flags, args: string[]) {
    super(flags, args)
  }

  makeRawTransaction = async () => {
    throw new Error('Query method does not have any tx')
  }

  fetchState = async (multisig: string, proposalId?: number): Promise<State> => {
    const query = this.provider.wasm.contractQuery.bind(this.provider.wasm)
    return fetchProposalState(query)(multisig, proposalId)
  }

  execute = async () => {
    const msig = this.args[0] || process.env.CW3_FLEX_MULTISIG
    const proposalId = Number(this.flags.proposal)
    const state = await this.fetchState(msig, proposalId)

    logger.log(state)
    return {} as Result<TransactionResponse>
  }
}

export const fetchProposalState = (query: (contractAddress: string, query: any) => Promise<any>) => async (
  multisig: string,
  proposalId?: number,
): Promise<State> => {
  const groupState = await query(multisig, {
    list_voters: {},
  })
  const owners = groupState.voters.map((m) => m.addr)
  const thresholdState = await query(multisig, {
    threshold: {},
  })
  const threshold = thresholdState.absolute_count.total_weight
  if (!proposalId) {
    return {
      threshold,
      nextAction: Action.CREATE,
      owners,
      approvers: [],
    }
  }
  const proposalState = await query(multisig, {
    proposal: {
      proposal_id: proposalId,
    },
  })
  const votes = await query(multisig, {
    list_votes: {
      proposal_id: proposalId,
    },
  })
  const status = proposalState.status
  const toNextAction = {
    passed: Action.EXECUTE,
    open: Action.APPROVE,
    pending: Action.APPROVE,
    rejected: Action.NONE,
    executed: Action.NONE,
  }

  return {
    threshold,
    nextAction: toNextAction[status],
    owners,
    currentStatus: status,
    data: proposalState.msgs,
    approvers: votes.votes.filter((v) => v.vote === Vote.YES).map((v) => v.voter),
    expiresAt: proposalState.expires.at_time ? new Date(proposalState.expires.at_time / 1e6) : null,
  }
}
