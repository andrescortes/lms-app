import { execFileSync } from 'child_process';
import * as fs from 'fs';
import * as path from 'path';

let DISPATCHER = '';
let INTERPRETER = '';

function canImportSpecifyCli(py: string): boolean {
  try {
    execFileSync(py, ['-c', 'import specify_cli'], {
      stdio: ['ignore', 'ignore', 'ignore'],
      timeout: 10000,
    });
    return true;
  } catch {
    return false;
  }
}

function resolveDispatcher(directory: string): void {
  DISPATCHER = path.join(directory, '.specify', 'events.py');
  const venvPy = path.join(directory, '.venv', 'bin', 'python');
  const venvWin = path.join(directory, '.venv', 'Scripts', 'python.exe');
  INTERPRETER = (
    (fs.existsSync(venvPy) && canImportSpecifyCli(venvPy) && venvPy) ||
    (fs.existsSync(venvWin) && canImportSpecifyCli(venvWin) && venvWin) ||
    (process.platform === 'win32' ? 'python' : 'python3')
  ) as string;
}

function runEvent(command: string, event: string, input: any, output: any, timeoutSec: number): string {
  if (!DISPATCHER) return '';
  try {
    return execFileSync(INTERPRETER, [DISPATCHER, command, event, String(timeoutSec)], {
      input: JSON.stringify({ input, output }),
      stdio: ['pipe', 'pipe', 'inherit'],
      encoding: 'utf-8',
      timeout: (timeoutSec + 5) * 1000,
    });
  } catch (e) {
    throw new Error(`specify event ${command} (${event}) failed: ${(e as Error).message}`);
  }
}

function _user_prompt_submit(input: any, output: any): string {
  const errors: string[] = [];
  const contexts: string[] = [];
  try {
    const ctx = runEvent('speckit.companion.on-prompt', 'user_prompt_submit', input, output, 10);
    if (ctx) contexts.push(ctx);
  } catch (e) {
    errors.push((e as Error).message);
  }
  if (errors.length > 0) {
    throw new Error(errors.join('; '));
  }
  return contexts.join('\n\n');
}

export default {
  id: 'speckit-events',
  async setup(ctx: any) {
    const directory = ctx?.location?.directory ?? ctx?.directory ?? process.cwd();
    resolveDispatcher(directory);

    await ctx.session.hook('prompt', async (event: any) => {
      const promptText =
        typeof event?.prompt === 'string'
          ? event.prompt
          : typeof event?.prompt?.text === 'string'
            ? event.prompt.text
            : '';

      const compatInput = {
        sessionID: event?.sessionID,
        prompt: event?.prompt,
      };
      const compatOutput = {
        parts: promptText ? [{ type: 'text', text: promptText }] : [],
      };

      const injectedContext = _user_prompt_submit(compatInput, compatOutput);
      if (!injectedContext) return;

      if (typeof ctx.session?.synthetic === 'function') {
        await ctx.session.synthetic({
          sessionID: event.sessionID,
          text: injectedContext,
          delivery: event.delivery,
          resume: false,
        });
      }
    });
  },
};