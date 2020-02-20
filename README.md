# tbot [![Build Status](https://github.com/tennuem/tbot/workflows/build/badge.svg)](https://github.com/tennuem/tbot/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/tennuem/tbot)](https://goreportcard.com/report/github.com/tennuem/tbot) [![Coverage Status](https://coveralls.io/repos/github/tennuem/tbot/badge.svg?branch=master)](https://coveralls.io/github/tennuem/tbot?branch=master)

## Configuration

| Command line   | Environment         | Default | Description     |
| -------------- | :------------------ | :------ | :-------------- |
| telegram.token | TBOT_TELEGRAM_TOKEN | string  | telegram token  |
| logger.level   | TBOT_LOGGER_LEVEL   | info    | level of logger |

## Usage

```bash
$ docker build --no-cache -t tbot .
$ docker run tbot
```
