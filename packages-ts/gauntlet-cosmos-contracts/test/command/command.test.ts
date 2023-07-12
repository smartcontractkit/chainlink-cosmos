import Cosmos from '../../src/commands'

describe('Command', () => {
  it('Load Cosmos commands', () => {
    expect(Cosmos.length).toBeGreaterThan(0)
  })
})
