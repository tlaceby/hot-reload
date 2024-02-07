// deno-lint-ignore-file require-await no-explicit-any

import * as path from "https://deno.land/std@0.214.0/path/mod.ts";
import * as fs from "https://deno.land/std@0.214.0/fs/mod.ts";

/* CLI Usage

hotreload init
  - Will create a file called .hotreload-settings.json file in supplied directory.

hotreload start
  -   Will look for the specified .hotreload-settings.json file
  and run the hotreload according to those specs.

*/

interface SettingsSchema {
  watchFileTypes?: string[],
  watchPaths?: string[],
  watchDelay?: number,
  commands: string[]
}

type ArgHandler = (params: string[]) => any;
const SETTINGS_FILE_NAME = ".hotreload-settings.json";
const ARG_HANDLERS = new Map<string, ArgHandler>();

ARG_HANDLERS.set("init", initHandler);
ARG_HANDLERS.set("start", startHandler);
ARG_HANDLERS.set("help", helpHandler);

Deno.exit(await main() ? 0: 1);

function error (error: string) {
  console.error(error);
  return false;
}

async function main () {
  const args = Deno.args.map(a => a.toLowerCase());

  if (args.length === 0) return helpHandler([]);

  for (const key of ARG_HANDLERS.keys()) {
    if (args[0] === key) {
      const remainingArgs = args.slice(1);
      return await (ARG_HANDLERS.get(key) as ArgHandler)(remainingArgs);
    }
  }

  return await helpHandler([]);
}

async function initHandler (flags: string[]) {
  const settingsFilePath = path.join(Deno.cwd(), SETTINGS_FILE_NAME);
  
  if (await fs.exists(settingsFilePath)) {
    if (flags.length === 0 || flags[0] !== "--force") {
      return error(`Settings file already exists. Use the --force flag to override the existing file: \n - ${settingsFilePath} already exists.`)
    }

    await Deno.remove(settingsFilePath);
  }

  const defaultSettings: SettingsSchema = {
    watchFileTypes: ["js", "ts", "css", "html"],
    watchPaths: ["."],
    watchDelay: 200,
    commands: ["echo \"Changes Made!\"", "echo \"Run your commands here.\""]
  }

  try {
    await Deno.writeTextFile(settingsFilePath, JSON.stringify(defaultSettings, null, 2), { createNew: true });
  } catch (err) {
    return error(err);
  }

  return true;
  //
}

async function helpHandler (_: string[]) {
  console.log("\n-----------------------------------\nAvailable Commands:\n");

  console.log("[init] Creates the .hotreload-settings.json inside the current directory.")
  console.log(" - (--force) Can be used to override an existing settings file.")
  
  console.log("\n[start] Looks for the .hotreload-settings.json file and will begin watching. \nThe settings.json file should be located inside the same directory the script is run from.")
  
  console.log("\n[help] Displays the help menu you can see now!\n-----------------------------------\n")
  
  return true;
}

async function startHandler (_: string[]) {
  const settingsFilePath = path.join(Deno.cwd(), SETTINGS_FILE_NAME);

  if (!fs.existsSync(settingsFilePath)) {
    return error(`Expected to find ${settingsFilePath} but it does not exist.`)
  }

  const data = await Deno.readTextFile(settingsFilePath);
  let settings: SettingsSchema;

  try {
    settings = JSON.parse(data); 
  } catch (_) {
    return error(`There was an issue parsing the JSON file. Please make sure it is valid.`)
  }

  let { watchFileTypes, watchPaths, commands, watchDelay } = settings;

  // Set defaults if noting is passed
  watchFileTypes ??= ["*"];
  watchPaths ??= ["."];
  watchDelay ??= 200;

  // Validate Commands
  let validCommands = true;

  try {
    const commandsArr = Array.from(commands);
    for (const command of commandsArr) {
      if (typeof command !== "string") validCommands = false;
    }
  } catch (_) {
    validCommands = false;
  }

  if (!validCommands) {
    return error(`Invalid ${SETTINGS_FILE_NAME} file. \n - "commands" should be a array of commands in string format.`)
  }

  const wg = [] as Promise<void>[];

  for (const pathToWatch of watchPaths) {
    wg.push(watchPath(path.join(Deno.cwd(), pathToWatch), commands, watchFileTypes, watchDelay));
  }

  return Promise.all(wg);
}

async function readDirRecursively (dir: string) {
  let paths = [] as string[];

  if (!fs.existsSync(dir)) return paths;

  if ((await Deno.lstat(dir)).isFile) {
    return [ dir ];
  }

  for await (const entry of Deno.readDir(dir)) {
    const loc = path.join(dir, entry.name);

    if (entry.isDirectory) paths = [...paths, ...(await readDirRecursively(loc))]
    else paths.push(loc);
  }

  return paths;
}

async function watchPath (watchPath: string, commands: string[], fileTypes: string[], watchDelay: number) {
  const files = new Map<string, string>();
  
  while (true) {
    let filesChanged = false;

    const paths = await readDirRecursively(watchPath);
    const checkedPaths = [] as string[];

    if (fileTypes.includes("*")) {
      for (const filePath of paths) {
        checkedPaths.push(filePath);
        const prevlastModified = files.get(filePath) ?? "";
        const lastModified = (await Deno.stat(filePath))?.mtime ?? new Date();
        const lastModfiedDate = lastModified.toLocaleString();

        if (lastModfiedDate !== prevlastModified) {
          filesChanged = true;
          files.set(filePath, lastModfiedDate);
        }
      }
    } else {
      for (const filePath of paths) {
        for (const ft of fileTypes) {
          const split = filePath.split(".");
          const ext = split[split.length - 1];
          
          if (ext && ext == ft) {
            checkedPaths.push(filePath);
            const prevlastModified = files.get(filePath) ?? "";
            const lastModified = (await Deno.stat(filePath))?.mtime ?? new Date();
            const lastModfiedDate = lastModified.toLocaleString();
  
            if (lastModfiedDate !== prevlastModified) {
              filesChanged = true;
              files.set(filePath, lastModfiedDate);
            }
          }
        }
      }
    }

    // Check all keys inside files and see if there are any which were not checked.
    // This handles deletes

    for (const filePath of files.keys()) {
      if (!checkedPaths.includes(filePath)) {
        files.delete(filePath);
        filesChanged = true;
        break;
      }
    }

    if (filesChanged) {
      for (const command of commands) {
        const process = Deno.run({
          cmd: ["/bin/sh", "-c", command], // Use `/bin/sh` with `-c` to interpret the command string
          stdout: "piped", // Capture standard output
        });

        const output = await process.output();
        console.log(new TextDecoder().decode(output));
      }
    }

    await sleep(watchDelay);
  }
}

async function sleep (ms: number) {
  return new Promise(r => setTimeout(r, ms));
}