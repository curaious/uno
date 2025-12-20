# Uno LLM Gateway CLI

Quickstart command for Uno LLM Gateway that can be run via `npx`.

## Usage

```bash
npx uno-llm-gateway
```

This command will:
1. Check for Docker and Docker Compose
2. Download required configuration files
3. Start all Uno services via Docker Compose

## Publishing

To publish this package to npm:

```bash
cd cli
npm publish
```

Make sure to:
1. Update the version in `package.json` before publishing
2. Test locally using `npm link` or `npm install -g .`
3. Ensure the GitHub repository URL in the script is correct

## Local Development

To test locally without publishing:

```bash
cd cli
npm install
npm link
uno-llm-gateway
```

Or run directly:

```bash
cd cli
npm install
node bin/uno-llm-gateway.js
```

