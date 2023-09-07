module.exports = {
  rootDir: '.',
  projects: [
    {
      displayName: 'gauntlet-cosmos',
      preset: 'ts-jest',
      testEnvironment: 'node',
      testMatch: ['<rootDir>/packages-ts/gauntlet-cosmos/**/*.test.ts'],
      globals: {
        'ts-jest': {
          tsconfig: '<rootDir>/packages-ts/gauntlet-cosmos/tsconfig.json',
        },
      },
    },
    {
      displayName: 'gauntlet-cosmos-contracts',
      preset: 'ts-jest',
      testEnvironment: 'node',
      testMatch: ['<rootDir>/packages-ts/gauntlet-cosmos-contracts/**/*.test.ts'],
      globals: {
        'ts-jest': {
          tsconfig: '<rootDir>/packages-ts/gauntlet-cosmos-contracts/tsconfig.json',
        },
      },
    },
    {
      displayName: 'gauntlet-cosmos-cw-plus',
      preset: 'ts-jest',
      testEnvironment: 'node',
      testMatch: ['<rootDir>/packages-ts/gauntlet-cosmos-cw-plus/**/*.test.ts'],
      globals: {
        'ts-jest': {
          tsconfig: '<rootDir>/packages-ts/gauntlet-cosmos-cw-plus/tsconfig.json',
        },
      },
    },
  ],
}
