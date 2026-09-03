import { cp, mkdir, readdir, rm } from "node:fs/promises";

const sourceDirectory = new URL(
  "../node_modules/@openproficiency/schema/schemas/",
  import.meta.url,
);
const destinationDirectory = new URL("../internal/schema/assets/", import.meta.url);

await mkdir(destinationDirectory, { recursive: true });

const sourceFiles = (await readdir(sourceDirectory))
  .filter((name) => name.endsWith(".schema.json"))
  .sort();
const destinationFiles = (await readdir(destinationDirectory))
  .filter((name) => name.endsWith(".schema.json"));

await Promise.all(
  destinationFiles.map((name) => rm(new URL(name, destinationDirectory))),
);
await Promise.all(
  sourceFiles.map((name) =>
    cp(new URL(name, sourceDirectory), new URL(name, destinationDirectory)),
  ),
);
