# Houdini LSP Development Workflow

## Quick Start

1. **Terminal 1: Run dev server**
   ```bash
   cd packages/houdini-lsp
   pnpm dev
   ```
   This watches for changes and rebuilds automatically with shebang.

2. **Terminal 2: Open Neovim**
   ```bash
   nvim test.graphql
   ```

3. **After making code changes:**
   - The dev server auto-rebuilds
   - In Neovim, run: `:LspRestart houdini_lsp`
   - Test your changes!

## Testing

### Check LSP is running
```vim
:LspInfo
```
Should show `houdini_lsp` attached.

### Test autocomplete
Go to line with `...UserAvatar @with(` and type inside the parentheses.
Press `<C-Space>` or `<C-x><C-o>` to trigger completion.

### Test diagnostics
Look for red underlines on queries missing required arguments:
```graphql
...UserAvatar @with(rounded: true)
# ^ Should error: Missing required argument 'width'
```

## Useful Neovim Commands

- `:LspInfo` - Check LSP status
- `:LspRestart houdini_lsp` - Restart after rebuilding
- `:LspLog` - View LSP logs for debugging
- `<leader>d` - Show diagnostic at cursor
- `K` - Hover (when implemented)
- `gd` - Go to definition (when implemented)

## Debug Mode

To see LSP communication:
```bash
# Set log level to DEBUG
export LSP_LOG_LEVEL=DEBUG
nvim test.graphql
```

Then check logs:
```vim
:LspLog
```

## Project Structure

```
src/
├── server.ts           # Main LSP entry point
├── types.ts            # Core type definitions
├── parser/
│   ├── argumentsParser.ts   # Parse @arguments directive
│   └── fragmentIndex.ts     # Fragment index
└── providers/
    ├── completion.ts        # Autocomplete for @with
    └── diagnostics.ts       # Validation errors
```

## Common Issues

### LSP not attaching
- Check `:LspInfo` shows houdini_lsp in "Enabled Configurations"
- Verify root_dir pattern matches your project
- Check `~/.local/state/nvim/lsp.log` for errors

### Changes not picked up
- Run `:LspRestart houdini_lsp` after rebuild
- Or close and reopen the file

### Server crashes
- Check `~/.local/state/nvim/lsp.log`
- Run `node dist/server.js --stdio` manually to see errors
