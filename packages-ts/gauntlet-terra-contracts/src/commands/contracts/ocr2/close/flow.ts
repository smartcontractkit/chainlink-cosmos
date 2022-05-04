import { FlowCommand } from '@chainlink/gauntlet-core'
import { waitExecute, TransactionResponse } from '@chainlink/gauntlet-terra'
import BeginProposal from '../proposal/beginProposal'
import FinalizeProposal from '../proposal/finalizeProposal'
import ProposeConfigClose from './proposeConfig'
import ProposeOffchainConfigClose from './proposeOffchainConfig'
import AcceptProposalClose from './acceptProposal'
import Inspect from '../inspection/inspect'
import { CATEGORIES } from '../../../../lib/constants'

export default class CloseFlow extends FlowCommand<TransactionResponse> {
  static id = 'ocr2:close:flow'
  static category = CATEGORIES.OCR
  static examples = ['yarn gauntlet ocr2:close:flow --network=local --rdd=<PATH_TO_RDD> <CONTRACT_ADDRESS>']

  constructor(flags, args) {
    super(flags, args, waitExecute)

    this.stepIds = {
      BEGIN_PROPOSAL: 2,
      FINALIZE_PROPOSAL: 3,
    }

    this.flow = [
      {
        id: this.stepIds.BEGIN_PROPOSAL,
        name: 'Begin Proposal',
        command: BeginProposal,
      },
      {
        name: 'Propose Config Close',
        command: ProposeConfigClose,
        flags: {
          configProposal: FlowCommand.ID.data(this.stepIds.BEGIN_PROPOSAL, 'proposalId'),
        },
      },
      {
        name: 'Propose Offchain Config Close',
        command: ProposeOffchainConfigClose,
        flags: {
          configProposal: FlowCommand.ID.data(this.stepIds.BEGIN_PROPOSAL, 'proposalId'),
        },
      },
      {
        id: this.stepIds.FINALIZE_PROPOSAL,
        name: 'Finalize Proposal',
        command: FinalizeProposal,
        flags: {
          configProposal: FlowCommand.ID.data(this.stepIds.BEGIN_PROPOSAL, 'proposalId'),
        },
      },
      {
        name: 'Accept Proposal Close',
        command: AcceptProposalClose,
        flags: {
          configProposal: FlowCommand.ID.data(this.stepIds.BEGIN_PROPOSAL, 'proposalId'),
          digest: FlowCommand.ID.data(this.stepIds.FINALIZE_PROPOSAL, 'digest'),
        },
      },
      // Inspection here
      {
        name: 'Inspection',
        command: Inspect,
      },
    ]
  }
}
