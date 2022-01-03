import Ajv from 'ajv'

import JTD from 'ajv/dist/jtd'

const ajv = new Ajv().addFormat('uint8', (value: any) => !isNaN(value))

export default ajv

const jtd = new JTD()
export { jtd }
