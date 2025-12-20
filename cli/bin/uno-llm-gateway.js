#!/usr/bin/env node

const fs = require('fs-extra');
const path = require('path');
const { execSync } = require('child_process');
const os = require('os');
const https = require('https');
const http = require('http');

const REPO_URL = 'https://raw.githubusercontent.com/praveen001/uno/main/deployments';
const FILES_TO_DOWNLOAD = [
  'docker-compose.yaml',
  'clickhouse-init.sql',
  'otel-collector-config.yaml'
];

async function checkDocker() {
  try {
    execSync('docker --version', { stdio: 'ignore' });
    execSync('docker compose version', { stdio: 'ignore' });
    return true;
  } catch (error) {
    console.error('❌ Docker and Docker Compose are required but not found.');
    console.error('Please install Docker Desktop: https://www.docker.com/products/docker-desktop/');
    process.exit(1);
  }
}

function downloadFile(url, filePath) {
  return new Promise((resolve, reject) => {
    const client = url.startsWith('https') ? https : http;
    const file = fs.createWriteStream(filePath);
    
    client.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // Handle redirects
        return downloadFile(response.headers.location, filePath).then(resolve).catch(reject);
      }
      
      if (response.statusCode !== 200) {
        file.close();
        if (fs.existsSync(filePath)) {
          fs.unlinkSync(filePath);
        }
        reject(new Error(`Failed to download ${url}: ${response.statusCode} ${response.statusMessage}`));
        return;
      }
      
      response.pipe(file);
      
      file.on('finish', () => {
        file.close();
        resolve();
      });
    }).on('error', (err) => {
      file.close();
      if (fs.existsSync(filePath)) {
        fs.unlinkSync(filePath);
      }
      reject(err);
    });
  });
}

async function downloadFiles(targetDir) {
  console.log('📥 Downloading required files...');
  
  for (const file of FILES_TO_DOWNLOAD) {
    const url = `${REPO_URL}/${file}`;
    const filePath = path.join(targetDir, file);
    try {
      await downloadFile(url, filePath);
      console.log(`  ✓ Downloaded ${file}`);
    } catch (error) {
      console.error(`❌ Failed to download ${file}:`, error.message);
      process.exit(1);
    }
  }
}

async function main() {
  console.log('🚀 Starting Uno LLM Gateway...\n');
  
  // Check Docker
  await checkDocker();
  console.log('✓ Docker and Docker Compose are available\n');
  
  // Create temporary directory
  const tempDir = path.join(os.tmpdir(), 'uno-llm-gateway');
  await fs.ensureDir(tempDir);
  
  // Download files
  await downloadFiles(tempDir);
  
  // Change to temp directory and run docker compose
  console.log('\n🐳 Starting Docker containers...\n');
  process.chdir(tempDir);
  
  try {
    execSync('docker compose up -d', { stdio: 'inherit' });
    console.log('\n✅ Uno LLM Gateway is starting up!');
    console.log('\n📊 Dashboard: http://localhost:3000');
    console.log('🔌 API: http://localhost:6060');
    console.log('\n💡 To stop: docker compose down');
    console.log(`💡 Files location: ${tempDir}`);
  } catch (error) {
    console.error('\n❌ Failed to start containers:', error.message);
    process.exit(1);
  }
}

main().catch(error => {
  console.error('Error:', error);
  process.exit(1);
});

