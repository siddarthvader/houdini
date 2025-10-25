import {
  createConnection,
  TextDocuments,
  ProposedFeatures,
  InitializeParams,
  CompletionParams,
  TextDocumentSyncKind,
  InitializeResult
} from 'vscode-languageserver/node';
import { TextDocument } from 'vscode-languageserver-textdocument';

import { FragmentIndexImpl } from './parser/fragmentIndex';
import { CompletionProvider } from './providers/completion';
import { DiagnosticsProvider } from './providers/diagnostics';

// Create LSP connection
const connection = createConnection(ProposedFeatures.al);

// TEST: Log immediately to see if server even starts
connection.console.error('!!!! HOUDINI LSP SERVER STARTING !!!!');

// Create document manager
const documents = new TextDocuments(TextDocument);

// Create fragment index (will pass connection later)
let fragmentIndex: FragmentIndexImpl;
let completionProvider: CompletionProvider;
let diagnosticsProvider: DiagnosticsProvider;

connection.onInitialize((params: InitializeParams): InitializeResult => {
  connection.console.error('🚀 Initializing Houdini Language Server');

  // Initialize with connection for logging
  fragmentIndex = new FragmentIndexImpl(connection);
  completionProvider = new CompletionProvider(fragmentIndex, connection);
  diagnosticsProvider = new DiagnosticsProvider(fragmentIndex);

  return {
    capabilities: {
      textDocumentSync: TextDocumentSyncKind.Incremental,
      completionProvider: {
        resolveProvider: false,
        triggerCharacters: ['(', ',', ' ']
      },
      hoverProvider: true,
      definitionProvider: true
    }
  };
});

// Index documents when they're opened or changed
documents.onDidOpen(e => {
  connection.console.error(`📄 Document opened: ${e.document.uri}`);
  fragmentIndex.indexDocument(e.document.uri, e.document.getText());
  validateDocument(e.document);
});

documents.onDidChangeContent(e => {
  connection.console.error(`✏️  Document changed: ${e.document.uri}`);
  fragmentIndex.indexDocument(e.document.uri, e.document.getText());
  validateDocument(e.document);
});

documents.onDidClose(e => {
  connection.console.error(`❌ Document closed: ${e.document.uri}`);
  fragmentIndex.removeByUri(e.document.uri);
});

// Provide completions
connection.onCompletion((params: CompletionParams) => {
  connection.console.error(`🔍 Completion requested at ${params.position.line}:${params.position.character}`);
  const document = documents.get(params.textDocument.uri);
  if (!document) {
    connection.console.error('❌ Document not found for completion');
    return null;
  }

  const result = completionProvider.provideCompletions(params, document.getText());
  connection.console.error(`💡 Returning ${result?.items?.length ?? 0} completion items`);
  return result;
});

// Validate and send diagnostics
function validateDocument(document: TextDocument): void {
  connection.console.error(`🔎 Validating document: ${document.uri}`);
  const diagnostics = diagnosticsProvider.validateDocument(
    document.uri,
    document.getText()
  );

  connection.console.error(`📊 Found ${diagnostics.length} diagnostics`);
  connection.sendDiagnostics({
    uri: document.uri,
    diagnostics
  });
}

// Listen on the connection
documents.listen(connection);
connection.listen();

