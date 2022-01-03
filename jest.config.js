module.exports = {
  rootDir: '.',
  projects: [
    {
      displayName: 'gauntlet-terra',
      preset: 'ts-jest',
      testEnvironment: 'node',
      testMatch: ['<rootDir>/packages-ts/gauntlet-terra/**/*.test.ts'],
      globals: {
        'ts-jest': {
          tsconfig: '<rootDir>/packages-ts/gauntlet-terra/tsconfig.json',
        },
      },
    },
    {
      displayName: 'gauntlet-terra-contracts',
      preset: 'ts-jest',
      testEnvironment: 'node',
      testMatch: ['<rootDir>/packages-ts/gauntlet-terra-contracts/**/*.test.ts'],
      globals: {
        'ts-jest': {
          tsconfig: '<rootDir>/packages-ts/gauntlet-terra-contracts/tsconfig.json',
        },
      },
    },
  ],
}
