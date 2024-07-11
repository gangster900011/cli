# bgammon-cli - Terminal-based client for [bgammon.org](https://bgammon.org)
[![Donate via LiberaPay](https://img.shields.io/liberapay/receives/rocket9labs.com.svg?logo=liberapay)](https://liberapay.com/rocket9labs.com)
[![Donate via Patreon](https://img.shields.io/badge/dynamic/json?color=%23e85b46&label=Patreon&query=data.attributes.patron_count&suffix=%20patrons&url=https%3A%2F%2Fwww.patreon.com%2Fapi%2Fcampaigns%2F5252223)](https://www.patreon.com/rocketnine)

## Demo

### Web

https://terminal.bgammon.org

### SSH

`ssh bgammon.org -p 5000`

## Installation

To install `bgammon-cli` to `~/go/bin/bgammon-cli`, execute the following command:

`go install code.rocket9labs.com/tslocum/bgammon-cli@latest`

## Usage

When starting `bgammon-cli`, provide your username and password:

`bgammon-cli --username MyAccount --password MySecretPassword`

**PROTIP:** bgammon.org supports using all available dice rolls to move a
checker with a single stroke. For instance, when you roll double 2s, you may
move a checker eight spaces by dragging it directly from space 2 to space 10.

### Keybindings

- `R` Roll
- `K` Ok (confirm moves, end turn)
- `Backspace` Undo move
- `Enter` Select, toggle input field focus

## Support

Please share issues and suggestions [here](https://code.rocket9labs.com/tslocum/bgammon-cli/issues).

For information on how to play backgammon visit https://bkgm.com/rules.html
