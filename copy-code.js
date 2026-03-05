#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const EXTENSIONS = new Set(['.go', '.sql']);
const SKIP_DIRS = new Set(['vendor', '.git', 'node_modules', 'dist', 'build', 'docs', 'migrations']);
const SKIP_FILES = new Set(['docs.go', 'migrate.go']);

function walk(dir) {
    let result = '';

    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
        const fullPath = path.join(dir, entry.name);

        if (entry.isDirectory()) {
            if (!SKIP_DIRS.has(entry.name)) result += walk(fullPath);
            continue;
        }

        if (!EXTENSIONS.has(path.extname(entry.name))) continue;
        if (SKIP_FILES.has(entry.name)) continue;

        const content = fs.readFileSync(fullPath, 'utf8');
        result += `// FILE: ${fullPath}\n\`\`\`\n${content}\n\`\`\`\n\n`;
    }

    return result;
}

const output = walk('.');

const sizeKB = (output.length / 1024).toFixed(0);
if (output.length > 200_000) {
    console.warn(`⚠️  Размер: ${sizeKB} KB — может быть слишком много`);
} else {
    console.log(`📦 Размер: ${sizeKB} KB`);
}

const commands = ['clip', 'pbcopy', 'xclip -selection clipboard'];

for (const cmd of commands) {
    try {
        execSync(cmd, { input: output });
        console.log('✅ Скопировано в буфер обмена!');
        process.exit(0);
    } catch {}
}

console.error('❌ Не удалось скопировать.');