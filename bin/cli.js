#!/usr/bin/env node

const { spawn, execSync } = require('child_process');
const path = require('path');
const fs = require('fs');

const DEPLOYMENTS_DIR = path.join(__dirname, '..', 'deployments');
const COMPOSE_FILE = path.join(DEPLOYMENTS_DIR, 'docker-compose.yaml');
const PACKAGE_JSON = path.join(__dirname, '..', 'package.json');

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
  
  // Read version from package.json and construct full image name
  let env = { ...process.env };
  if (fs.existsSync(PACKAGE_JSON)) {
    try {
      const pkg = JSON.parse(fs.readFileSync(PACKAGE_JSON, 'utf8'));
      if (pkg.version && !env.UNO_IMAGE) {
        // Only set UNO_IMAGE if it's not already set (allows manual override)
        env.UNO_IMAGE = `praveenraj9495/uno-gateway:${pkg.version}`;
        log(`${DIM}Using Uno version: ${pkg.version}${RESET}\n`);
      }
    } catch (err) {
      // If we can't read package.json, continue without version
    }
  }
  
  const proc = spawn('docker', composeArgs, {
    stdio: 'inherit',
    cwd: DEPLOYMENTS_DIR,
    env: env,
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

function parseArgs() {
  const args = process.argv.slice(2);
  const profiles = [];
  const composeArgs = [];
  let showHelp = false;
  
  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    
    if (arg === '--help' || arg === '-h') {
      showHelp = true;
    } else if (arg === '--temporal') {
      profiles.push('temporal');
    } else if (arg === '--restate') {
      profiles.push('restate');
    } else if (arg === '--all') {
      profiles.push('all');
    } else {
      // Pass through other arguments to docker compose
      composeArgs.push(arg);
    }
  }
  
  return { profiles, composeArgs, showHelp };
}

function showHelpMessage() {
  console.log(`
${BOLD}Usage:${RESET} npx @curaious/uno [options] [docker-compose-args]

${BOLD}Options:${RESET}
  --temporal                  Enable temporal worker profile
  --restate                  Enable restate service profile
  --all                      Enable all profiles (temporal + restate)
  --help, -h                 Show this help message

${BOLD}Profiles:${RESET}
  temporal                   Start temporal worker service
  restate                    Start restate service
  all                        Start both temporal and restate services

${BOLD}Examples:${RESET}
  npx @curaious/uno                    # Start default services (agent-server)
  npx @curaious/uno --temporal         # Start with temporal worker
  npx @curaious/uno --restate          # Start with restate service
  npx @curaious/uno --all              # Start with both temporal and restate
  npx @curaious/uno --temporal up -d   # Start temporal in detached mode

${BOLD}Note:${RESET}
  Any additional arguments are passed directly to docker compose.
  Use 'docker compose' commands like: up, down, ps, logs, etc.
`);
}

async function main() {
  const { profiles, composeArgs, showHelp } = parseArgs();
  
  if (showHelp) {
    showHelpMessage();
    process.exit(0);
  }
  
  banner();
  
  if (!checkDocker()) {
    process.exit(1);
  }
  
  if (!fs.existsSync(COMPOSE_FILE)) {
    log(`Error: docker-compose.yaml not found at ${COMPOSE_FILE}`, RED);
    process.exit(1);
  }

  // Build docker compose arguments
  const dockerArgs = [];
  
  // Add profile flags if any profiles are specified
  if (profiles.length > 0) {
    for (const profile of profiles) {
      dockerArgs.push('--profile', profile);
    }
    log(`Using profiles: ${profiles.join(', ')}`, CYAN);
  }
  
  // Add compose command (default to 'up' if no command provided)
  if (composeArgs.length === 0) {
    dockerArgs.push('up');
    log('Starting Uno services...', GREEN);
    log(`${DIM}This may take a few minutes on first run to pull images.${RESET}`);
  } else {
    dockerArgs.push(...composeArgs);
  }
  
  runDockerCompose(dockerArgs);
}

main().catch((err) => {
  log(`Error: ${err.message}`, RED);
  process.exit(1);
});

