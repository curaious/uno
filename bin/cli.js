#!/usr/bin/env node

const { spawn, execSync } = require('child_process');
const path = require('path');
const fs = require('fs');

const DEPLOYMENTS_DIR = path.join(__dirname, '..', 'deployments');
const COMPOSE_FILE = path.join(DEPLOYMENTS_DIR, 'docker-compose.yaml');

const CYAN = '\x1b[36m';
const GREEN = '\x1b[32m';
const YELLOW = '\x1b[33m';
const RED = '\x1b[31m';
const RESET = '\x1b[0m';
const BOLD = '\x1b[1m';
const DIM = '\x1b[2m';

function log(message, color = RESET) {
  console.log(`${color}${message}${RESET}`);
}

function banner() {
  console.log(`
${CYAN}${BOLD}
  ██╗   ██╗███╗   ██╗ ██████╗ 
  ██║   ██║████╗  ██║██╔═══██╗
  ██║   ██║██╔██╗ ██║██║   ██║
  ██║   ██║██║╚██╗██║██║   ██║
  ╚██████╔╝██║ ╚████║╚██████╔╝
   ╚═════╝ ╚═╝  ╚═══╝ ╚═════╝ 
${RESET}
  ${DIM}AI Gateway & Agent Framework${RESET}
`);
}

function checkDocker() {
  try {
    execSync('docker --version', { stdio: 'pipe' });
    execSync('docker compose version', { stdio: 'pipe' });
    return true;
  } catch (error) {
    log('Error: Docker and Docker Compose are required to run Uno.', RED);
    log('Please install Docker Desktop: https://www.docker.com/products/docker-desktop/', YELLOW);
    return false;
  }
}

function runDockerCompose(args, options = {}) {
  const composeArgs = [
    'compose',
    '-f', COMPOSE_FILE,
    ...args
  ];
  
  log(`\n${DIM}Running: docker ${composeArgs.join(' ')}${RESET}\n`);
  
  const proc = spawn('docker', composeArgs, {
    stdio: 'inherit',
    cwd: DEPLOYMENTS_DIR,
    ...options
  });
  
  proc.on('error', (err) => {
    log(`Failed to start docker compose: ${err.message}`, RED);
    process.exit(1);
  });
  
  proc.on('exit', (code) => {
    process.exit(code || 0);
  });
}

async function main() {
  banner();
  
  if (!checkDocker()) {
    process.exit(1);
  }
  
  if (!fs.existsSync(COMPOSE_FILE)) {
    log(`Error: docker-compose.yaml not found at ${COMPOSE_FILE}`, RED);
    process.exit(1);
  }

    log('Starting Uno services...', GREEN);
    log(`${DIM}This may take a few minutes on first run to pull images.${RESET}`);
    runDockerCompose(['up']);
}

main().catch((err) => {
  log(`Error: ${err.message}`, RED);
  process.exit(1);
});

