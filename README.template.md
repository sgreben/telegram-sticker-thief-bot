# ${APP}

Sticker thieving bot for Telegram. Send it stickers to steal them :) Public instance: [@the_sticker_thief_bot](https://t.me/the_sticker_thief_bot)

## Contents

- [Usage](#usage)
  - [Telegram](#telegram)
  - [CLI](#cli)
- [Get it](#get-it)

## Usage

### Telegram

- `/help`: Print help
- `/start`: Create your scratchpad sticker set
- `/list` : List scratchpad stickers
- `/clear`: Clear scratchpad sticker set
- `/clone` `[STICKER_SET]`: Make a permanent clone of the scratchpad sticker set, or the specified sticker set
- `/steal` `STICKER_SET` - Add all stickers from this set to the scratchpad sticker set
- `/zip` `[STICKER_SET]`: Download the scratchpad sticker set, or the specified sticker set as a zip archive

### CLI

```text
${APP} -token BOT_TOKEN

${USAGE}
```

## Get it

Using go get:

```bash
go get -u github.com/sgreben/${APP}
```

Or [download the binary for your platform](https://github.com/sgreben/${APP}/releases/latest) from the releases page.
