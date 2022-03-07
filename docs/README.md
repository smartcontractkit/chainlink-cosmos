# Chainlink Terra Integration

This repository is a monorepo of the various components required for Chainlink on Terra.

- [Terra Contracts](../contracts)
- [Terra CL Relay](../pkg/terra)
- [Terra Gauntlet](../packages-ts)
- [Terra On-chain Monitoring](../pkg/monitoring)
- [Ops](../ops)
- [Integration/E2E Tests](../tests/e2e)
- [Demos & Examples](../examples)

# Local asdf initial setup

    asdf plugin-add golang https://github.com/kennyp/asdf-golang.git 
    # for other golang requirements for your os go to https://github.com/kennyp/asdf-golang
    asdf plugin add nodejs https://github.com/asdf-vm/asdf-nodejs.git

    # Then run
    asdf install
