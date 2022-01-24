import Ajv from 'ajv'

import JTD from 'ajv/dist/jtd'

const ajv = new Ajv().addFormat('uint8', (value: any) => !isNaN(value))

ajv.addFormat('uint64', {
  type: 'number',
  validate: (x) => !isNaN(x),
})

export default ajv

const jtd = new JTD()
export { jtd }
