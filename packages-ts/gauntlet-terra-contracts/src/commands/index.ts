import Upload from './tooling/upload'
import TransferLink from './contracts/link/transfer'
import DeployLink from './contracts/link/deploy'
import OCR2 from './contracts/ocr2'
import ProxyOCR2 from './contracts/proxyOcr2'

export default [Upload, DeployLink, TransferLink, ...OCR2, ...ProxyOCR2]
