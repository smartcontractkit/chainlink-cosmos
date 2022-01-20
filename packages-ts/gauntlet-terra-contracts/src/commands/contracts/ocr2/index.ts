import SetupFlow from './setup.dev.flow'
import OCR2InitializeFlow from './initialize.flow'
import Deploy from './deploy'
import SetBilling from './setBilling'
import SetConfig from './setConfig'
import SetPayees from './setPayees'
import Inspect from './inspection/inspect'

export default [SetupFlow, Deploy, SetBilling, SetConfig, SetPayees, OCR2InitializeFlow, Inspect]
