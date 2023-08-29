import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing'

// script to generate random addresses
const printRandomAddrs = async (num: number) => {
  let testAddrs = await Promise.all(
    Array.from({ length: num }, async () => {
      const w = await DirectSecp256k1HdWallet.generate(12, { prefix: 'wasm' })
      const accs = await w.getAccounts()

      return accs[0].address
    }),
  )

  testAddrs.forEach((account, index) => {
    console.log(`Address #${index}`, account)
  })
}

printRandomAddrs(15)
