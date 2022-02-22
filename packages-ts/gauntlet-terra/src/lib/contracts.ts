import { TerraABI } from './schema'

export enum CONTRACT_LIST { }  // Placeholder, to be filled in by terra-gauntlet-contracts pkg
export type ContractId = keyof typeof CONTRACT_LIST

export type Contract = {
  id: ContractId
  abi: TerraABI
  bytecode: string
}

export type Contracts = Record<ContractId, Contract>

export type GetContract = (id: ContractId, version:string) => Promise<Contract>

const isValidContract = (contractName: string): boolean => {
  // Validate that we have this contract available
  return Object.values(CONTRACT_LIST).includes(contractName as CONTRACT_LIST)
}
