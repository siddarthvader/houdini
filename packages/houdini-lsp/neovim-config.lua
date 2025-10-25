-- Houdini LSP Configuration for Neovim
-- Copy this to your Neovim config (e.g., ~/.config/nvim/lua/lsp/houdini.lua)

local lspconfig = require('lspconfig')
local configs = require('lspconfig.configs')

-- Define Houdini LSP config if not already defined
if not configs.houdini_lsp then
  configs.houdini_lsp = {
    default_config = {
      cmd = { 'houdini-language-server', '--stdio' },
      filetypes = { 'graphql', 'typescript', 'typescriptreact', 'javascript', 'javascriptreact', 'svelte' },
      root_dir = lspconfig.util.root_pattern(
        '.graphqlrc.yaml',
        'houdini.config.js',
        'houdini.config.ts',
        'package.json'
      ),
      settings = {},
      init_options = {}
    },
    docs = {
      description = [[
Houdini Language Server for @arguments and @with directives.
Provides autocomplete and validation for Houdini fragment arguments.
]],
    }
  }
end

-- Setup the LSP
lspconfig.houdini_lsp.setup {
  on_attach = function(client, bufnr)
    -- Enable completion triggered by <c-x><c-o>
    vim.api.nvim_buf_set_option(bufnr, 'omnifunc', 'v:lua.vim.lsp.omnifunc')

    -- Keybindings
    local opts = { noremap = true, silent = true, buffer = bufnr }
    vim.keymap.set('n', 'gd', vim.lsp.buf.definition, opts)
    vim.keymap.set('n', 'K', vim.lsp.buf.hover, opts)
    vim.keymap.set('n', '<leader>ca', vim.lsp.buf.code_action, opts)
    vim.keymap.set('n', '[d', vim.diagnostic.goto_prev, opts)
    vim.keymap.set('n', ']d', vim.diagnostic.goto_next, opts)

    print("Houdini LSP attached to buffer " .. bufnr)
  end,
  -- If you use nvim-cmp for autocompletion
  capabilities = require('cmp_nvim_lsp').default_capabilities(),

  -- Or if you don't use nvim-cmp, use default capabilities
  -- capabilities = vim.lsp.protocol.make_client_capabilities(),
}
