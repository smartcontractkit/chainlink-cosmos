import Ajv from 'ajv'
import JTD from 'ajv/dist/jtd'
import { JSONSchemaType } from 'ajv'

const ajv = new Ajv().addFormat('uint8', (value: any) => !isNaN(value))

ajv.addFormat('uint64', {
  type: 'number',
  validate: (x) => !isNaN(x),
})

ajv.addFormat('uint32', {
  type: 'number',
  validate: (x) => !isNaN(x),
})

export enum TERRA_OPERATIONS {
  DEPLOY = 'instantiate',
  EXECUTE = 'execute',
  QUERY = 'query',
}

export type TerraABI = {
  [TERRA_OPERATIONS.DEPLOY]: JSONSchemaType<any>
  [TERRA_OPERATIONS.EXECUTE]: JSONSchemaType<any>
  [TERRA_OPERATIONS.QUERY]: JSONSchemaType<any>
}

export const isQueryFunction = (abi: TerraABI, functionName: string) => {
  const functionList = abi.query.oneOf || abi.query.anyOf
  return functionList.find((queryAbi: any) => {
    if (queryAbi.enum) return queryAbi.enum.includes(functionName)
    if (queryAbi.required) return queryAbi.required.includes(functionName)
    return false
  })
}

export default ajv

const jtd = new JTD()
export { jtd }
