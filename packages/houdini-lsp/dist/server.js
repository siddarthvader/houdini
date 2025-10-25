#!/usr/bin/env node
"use strict";

// src/server.ts
var import_node = require("vscode-languageserver/node");
var import_vscode_languageserver_textdocument = require("vscode-languageserver-textdocument");

// src/parser/fragmentIndex.ts
var import_graphql = require("graphql");

// src/parser/argumentsParser.ts
function parseArgumentsDirective(directive) {
  console.log("[ArgumentsParser] Parsing @arguments directive");
  const args = [];
  if (!directive.arguments) {
    console.log("[ArgumentsParser] No arguments in directive");
    return args;
  }
  console.log(`[ArgumentsParser] Found ${directive.arguments.length} directive arguments`);
  for (const arg of directive.arguments) {
    if (arg.value.kind !== "ObjectValue") {
      console.log(`[ArgumentsParser] Skipping non-object argument: ${arg.name.value}`);
      continue;
    }
    const config = parseArgumentConfig(arg.value);
    args.push({
      name: arg.name.value,
      type: config.type.replace("!", ""),
      required: config.type.endsWith("!"),
      default: config.default,
      location: {
        line: arg.loc?.startToken.line ?? 0,
        column: arg.loc?.startToken.column ?? 0
      }
    });
    console.log(`[ArgumentsParser] Parsed argument: ${arg.name.value} (${config.type}${config.default !== void 0 ? `, default: ${config.default}` : ""})`);
  }
  console.log(`[ArgumentsParser] Total parsed: ${args.length} arguments`);
  return args;
}
function parseArgumentConfig(objectValue) {
  let type = "";
  let defaultValue = void 0;
  for (const field of objectValue.fields) {
    if (field.name.value === "type") {
      if (field.value.kind === "StringValue") {
        type = field.value.value;
      }
    } else if (field.name.value === "default") {
      defaultValue = extractValue(field.value);
    }
  }
  return { type, default: defaultValue };
}
function extractValue(valueNode) {
  switch (valueNode.kind) {
    case "IntValue":
      return parseInt(valueNode.value, 10);
    case "FloatValue":
      return parseFloat(valueNode.value);
    case "StringValue":
      return valueNode.value;
    case "BooleanValue":
      return valueNode.value;
    case "NullValue":
      return null;
    default:
      return void 0;
  }
}

// src/parser/fragmentIndex.ts
var FragmentIndexImpl = class {
  constructor(connection2) {
    this.connection = connection2;
    this.fragments = /* @__PURE__ */ new Map();
  }
  /**
   * Parse a document and index all fragments
   */
  indexDocument(uri, content) {
    if (!uri.endsWith(".graphql") && !uri.endsWith(".gql")) {
      this.connection?.console.error(`[FragmentIndex] Skipping non-GraphQL file: ${uri}`);
      return;
    }
    try {
      const document = (0, import_graphql.parse)(content, { noLocation: false });
      this.removeByUri(uri);
      let fragmentCount = 0;
      for (const definition of document.definitions) {
        if (definition.kind === "FragmentDefinition") {
          const fragment = this.parseFragment(definition, uri);
          if (fragment) {
            this.set(fragment);
            fragmentCount++;
            this.connection?.console.error(`[FragmentIndex] Indexed fragment: ${fragment.name} with ${fragment.arguments.length} arguments`);
          }
        }
      }
      this.connection?.console.error(`[FragmentIndex] Total fragments indexed: ${fragmentCount}`);
      this.connection?.console.error(`[FragmentIndex] All fragments in index: ${Array.from(this.fragments.keys()).join(", ")}`);
    } catch (error) {
      this.connection?.console.error(`[FragmentIndex] Failed to parse document ${uri}: ${error}`);
    }
  }
  /**
   * Parse a fragment definition node
   */
  parseFragment(node, uri) {
    const argumentsDirective = node.directives?.find(
      (d) => d.name.value === "arguments"
    );
    const args = argumentsDirective ? parseArgumentsDirective(argumentsDirective) : [];
    return {
      name: node.name.value,
      typeCondition: node.typeCondition.name.value,
      arguments: args,
      uri,
      location: {
        line: node.loc?.startToken.line ?? 0,
        column: node.loc?.startToken.column ?? 0
      }
    };
  }
  set(fragment) {
    this.fragments.set(fragment.name, fragment);
  }
  get(name) {
    return this.fragments.get(name);
  }
  removeByUri(uri) {
    for (const [name, fragment] of this.fragments.entries()) {
      if (fragment.uri === uri) {
        this.fragments.delete(name);
      }
    }
  }
  getAll() {
    return Array.from(this.fragments.values());
  }
};

// src/providers/completion.ts
var import_vscode_languageserver = require("vscode-languageserver");
var CompletionProvider = class {
  constructor(fragmentIndex2, connection2) {
    this.fragmentIndex = fragmentIndex2;
    this.connection = connection2;
  }
  /**
   * Provide completions for @with directive arguments
   */
  provideCompletions(params, documentText) {
    const { position } = params;
    const lines = documentText.split("\n");
    const line = lines[position.line];
    const textBeforeCursor = line.substring(0, position.character);
    this.connection?.console.error(`[Completion] Text before cursor: "${textBeforeCursor}"`);
    const fragmentName = this.getFragmentNameForWith(textBeforeCursor, documentText);
    if (!fragmentName) {
      this.connection?.console.error("[Completion] No fragment name found");
      return null;
    }
    this.connection?.console.error(`[Completion] Fragment: ${fragmentName}`);
    const allFragments = this.fragmentIndex.getAll();
    this.connection?.console.error(`[Completion] Available fragments: ${allFragments.map((f) => f.name).join(", ")}`);
    const fragment = this.fragmentIndex.get(fragmentName);
    if (!fragment) {
      this.connection?.console.error(`[Completion] Fragment "${fragmentName}" not in index`);
      this.connection?.console.error(`[Completion] Available fragments: ${Array.from(this.fragmentIndex.fragments.keys()).join(", ")}`);
      return null;
    }
    if (fragment.arguments.length === 0) {
      this.connection?.console.error(`[Completion] Fragment "${fragmentName}" has no arguments`);
      return null;
    }
    this.connection?.console.error(`[Completion] Found ${fragment.arguments.length} arguments: ${fragment.arguments.map((a) => a.name).join(", ")}`);
    const items = fragment.arguments.map(
      (arg) => this.createCompletionItem(arg)
    );
    return {
      isIncomplete: false,
      items
    };
  }
  /**
   * Extract fragment name from @with context
   * Example: "...UserAvatar @with(" -> "UserAvatar"
   */
  getFragmentNameForWith(textBeforeCursor, fullText) {
    const match = textBeforeCursor.match(/\.\.\.(\w+)\s+@with\s*\(/);
    if (match) {
      return match[1];
    }
    return null;
  }
  /**
   * Create completion item for a fragment argument
   */
  createCompletionItem(arg) {
    const typeDisplay = arg.required ? `${arg.type}!` : arg.type;
    let documentation = `Type: ${typeDisplay}`;
    if (arg.default !== void 0) {
      documentation += `
Default: ${JSON.stringify(arg.default)}`;
    }
    if (arg.required) {
      documentation += "\n\u26A0\uFE0F Required argument";
    }
    return {
      label: arg.name,
      kind: import_vscode_languageserver.CompletionItemKind.Property,
      detail: typeDisplay,
      documentation: {
        kind: "markdown",
        value: documentation
      },
      insertText: `${arg.name}: `,
      sortText: arg.required ? `0_${arg.name}` : `1_${arg.name}`
      // Required args first
    };
  }
};

// src/providers/diagnostics.ts
var import_vscode_languageserver2 = require("vscode-languageserver");
var import_graphql2 = require("graphql");
var DiagnosticsProvider = class {
  constructor(fragmentIndex2) {
    this.fragmentIndex = fragmentIndex2;
  }
  /**
   * Validate @with directives and return diagnostics
   */
  validateDocument(uri, content) {
    const diagnostics = [];
    try {
      const document = (0, import_graphql2.parse)(content, { noLocation: false });
      this.visitDocument(document, (spread) => {
        const withDirective = spread.directives?.find(
          (d) => d.name.value === "with"
        );
        if (withDirective) {
          const errors = this.validateWithDirective(
            spread.name.value,
            withDirective
          );
          diagnostics.push(...errors);
        }
      });
    } catch (error) {
    }
    return diagnostics;
  }
  /**
   * Validate a single @with directive against its fragment definition
   */
  validateWithDirective(fragmentName, directive) {
    const diagnostics = [];
    const fragment = this.fragmentIndex.get(fragmentName);
    if (!fragment) {
      return diagnostics;
    }
    const providedArgs = new Set(
      directive.arguments?.map((a) => a.name.value) ?? []
    );
    for (const arg of fragment.arguments) {
      if (arg.required && !providedArgs.has(arg.name)) {
        diagnostics.push({
          severity: import_vscode_languageserver2.DiagnosticSeverity.Error,
          range: this.getRange(directive),
          message: `Missing required argument '${arg.name}' of type ${arg.type}!`,
          source: "houdini-lsp"
        });
      }
    }
    const validArgNames = new Set(fragment.arguments.map((a) => a.name));
    for (const providedArg of directive.arguments ?? []) {
      if (!validArgNames.has(providedArg.name.value)) {
        diagnostics.push({
          severity: import_vscode_languageserver2.DiagnosticSeverity.Warning,
          range: this.getRange(providedArg),
          message: `Unknown argument '${providedArg.name.value}' for fragment ${fragmentName}`,
          source: "houdini-lsp"
        });
      }
    }
    return diagnostics;
  }
  /**
   * Get range from a GraphQL AST node
   */
  getRange(node) {
    return {
      start: {
        line: (node.loc?.startToken.line ?? 1) - 1,
        // LSP is 0-indexed
        character: node.loc?.startToken.column ?? 0
      },
      end: {
        line: (node.loc?.endToken.line ?? 1) - 1,
        character: node.loc?.endToken.column ?? 0
      }
    };
  }
  /**
   * Visit all fragment spreads in the document
   */
  visitDocument(document, callback) {
    const visit = (node) => {
      if (node.kind === "FragmentSpread") {
        callback(node);
      }
      for (const key in node) {
        const value = node[key];
        if (Array.isArray(value)) {
          value.forEach(visit);
        } else if (value && typeof value === "object") {
          visit(value);
        }
      }
    };
    document.definitions.forEach(visit);
  }
};

// src/server.ts
var connection = (0, import_node.createConnection)(import_node.ProposedFeatures.al);
connection.console.error("!!!! HOUDINI LSP SERVER STARTING !!!!");
var documents = new import_node.TextDocuments(import_vscode_languageserver_textdocument.TextDocument);
var fragmentIndex;
var completionProvider;
var diagnosticsProvider;
connection.onInitialize((params) => {
  connection.console.error("\u{1F680} Initializing Houdini Language Server");
  fragmentIndex = new FragmentIndexImpl(connection);
  completionProvider = new CompletionProvider(fragmentIndex, connection);
  diagnosticsProvider = new DiagnosticsProvider(fragmentIndex);
  return {
    capabilities: {
      textDocumentSync: import_node.TextDocumentSyncKind.Incremental,
      completionProvider: {
        resolveProvider: false,
        triggerCharacters: ["(", ",", " "]
      },
      hoverProvider: true,
      definitionProvider: true
    }
  };
});
documents.onDidOpen((e) => {
  connection.console.error(`\u{1F4C4} Document opened: ${e.document.uri}`);
  fragmentIndex.indexDocument(e.document.uri, e.document.getText());
  validateDocument(e.document);
});
documents.onDidChangeContent((e) => {
  connection.console.error(`\u270F\uFE0F  Document changed: ${e.document.uri}`);
  fragmentIndex.indexDocument(e.document.uri, e.document.getText());
  validateDocument(e.document);
});
documents.onDidClose((e) => {
  connection.console.error(`\u274C Document closed: ${e.document.uri}`);
  fragmentIndex.removeByUri(e.document.uri);
});
connection.onCompletion((params) => {
  connection.console.error(`\u{1F50D} Completion requested at ${params.position.line}:${params.position.character}`);
  const document = documents.get(params.textDocument.uri);
  if (!document) {
    connection.console.error("\u274C Document not found for completion");
    return null;
  }
  const result = completionProvider.provideCompletions(params, document.getText());
  connection.console.error(`\u{1F4A1} Returning ${result?.items?.length ?? 0} completion items`);
  return result;
});
function validateDocument(document) {
  connection.console.error(`\u{1F50E} Validating document: ${document.uri}`);
  const diagnostics = diagnosticsProvider.validateDocument(
    document.uri,
    document.getText()
  );
  connection.console.error(`\u{1F4CA} Found ${diagnostics.length} diagnostics`);
  connection.sendDiagnostics({
    uri: document.uri,
    diagnostics
  });
}
documents.listen(connection);
connection.listen();
