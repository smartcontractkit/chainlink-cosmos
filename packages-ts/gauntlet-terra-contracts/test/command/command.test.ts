import Terra from '../../src/commands'

describe('Command', () => {
  it('Load Terra commands', () => {
    expect(Terra.length).toBeGreaterThan(0)
  })
})
