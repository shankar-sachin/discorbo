const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');

const ROOT_DIR = path.join(__dirname, '..', '..');
const SRC_DIR = path.join(__dirname, '..');
const ENGINE_MAIN = path.join(SRC_DIR, 'go', 'maze_engine', 'main.go');
const ENGINE_BIN_DIR = path.join(SRC_DIR, 'go', 'bin');
const ENGINE_BIN = path.join(ENGINE_BIN_DIR, process.platform === 'win32' ? 'maze_engine.exe' : 'maze_engine');

let buildPromise = null;

function findGoExe() {
  const candidates = [
    process.env.GO_EXE,
    process.env.GOROOT ? path.join(process.env.GOROOT, 'bin', process.platform === 'win32' ? 'go.exe' : 'go') : null,
    'C:\\Program Files\\Go\\bin\\go.exe',
    'go'
  ].filter(Boolean);

  for (const candidate of candidates) {
    if (candidate === 'go') {
      return candidate;
    }
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }

  return null;
}

function runProcess(command, args, stdinText) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      cwd: ROOT_DIR,
      windowsHide: true
    });

    let stdout = '';
    let stderr = '';

    child.stdout.on('data', chunk => { stdout += chunk.toString(); });
    child.stderr.on('data', chunk => { stderr += chunk.toString(); });
    child.on('error', reject);
    child.on('close', code => {
      if (code === 0) {
        resolve(stdout);
      } else {
        reject(new Error(stderr || `Process exited with code ${code}`));
      }
    });

    child.stdin.write(stdinText);
    child.stdin.end();
  });
}

async function ensureMazeEngineReady() {
  if (fs.existsSync(ENGINE_BIN)) {
    return ENGINE_BIN;
  }

  if (buildPromise) {
    return buildPromise;
  }

  buildPromise = (async () => {
    const goExe = findGoExe();
    if (!goExe) {
      throw new Error('Go is not installed or not reachable.');
    }

    if (!fs.existsSync(ENGINE_MAIN)) {
      throw new Error(`Maze engine source not found at ${ENGINE_MAIN}`);
    }

    fs.mkdirSync(ENGINE_BIN_DIR, { recursive: true });
    await runProcess(goExe, ['build', '-o', ENGINE_BIN, ENGINE_MAIN], '');
    return ENGINE_BIN;
  })();

  try {
    return await buildPromise;
  } finally {
    buildPromise = null;
  }
}

async function runMazeStep(state, direction) {
  const payload = JSON.stringify({
    op: 'step',
    direction,
    state
  });

  const engineBinary = await ensureMazeEngineReady();
  const stdout = await runProcess(engineBinary, [], payload);

  let parsed;
  try {
    parsed = JSON.parse(stdout);
  } catch {
    throw new Error('Maze engine returned invalid JSON.');
  }

  if (parsed.error) {
    throw new Error(parsed.error);
  }

  if (!parsed.state) {
    throw new Error('Maze engine returned no state.');
  }

  return parsed.state;
}

module.exports = {
  ensureMazeEngineReady,
  runMazeStep
};

