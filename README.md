# Chainlink Terra Integration

This repository is a monorepo of the various components required for Chainlink on Terra.

- Terra Contracts (OCR2, ...)
- Terra CL Relay
- Terra Gauntlet
- Terra On-chain Monitoring
- Ops (infrastructure)
- Integration (tests)
- Demos & Examples

# Local asdf initial setup

    asdf plugin-add golang https://github.com/kennyp/asdf-golang.git 
    # for other golang requirements for your os go to https://github.com/kennyp/asdf-golang
    asdf plugin add nodejs https://github.com/asdf-vm/asdf-nodejs.git
    asdf plugin-add rust https://github.com/asdf-community/asdf-rust.git 

    # Then run
    asdf install
