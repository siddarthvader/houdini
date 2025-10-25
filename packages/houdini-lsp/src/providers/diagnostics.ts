import {
  Diagnostic,
  DiagnosticSeverity,
  Range
} from 'vscode-languageserver';
import { parse, DocumentNode, FragmentSpreadNode, DirectiveNode } from 'graphql';
import { FragmentIndex } from '../types';

export class DiagnosticsProvider {
  constructor(private fragmentIndex: FragmentIndex) {}

  /**
   * Validate @with directives and return diagnostics
   */
  public validateDocument(uri: string, content: string): Diagnostic[] {
    const diagnostics: Diagnostic[] = [];

    try {
      const document = parse(content, { noLocation: false });

      // Walk the AST and find all fragment spreads with @with
      this.visitDocument(document, (spread) => {
        const withDirective = spread.directives?.find(
          d => d.name.value === 'with'
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
      // Parse error - will be reported by GraphQL LSP
    }

    return diagnostics;
  }

  /**
   * Validate a single @with directive against its fragment definition
   */
  private validateWithDirective(
    fragmentName: string,
    directive: DirectiveNode
  ): Diagnostic[] {
    const diagnostics: Diagnostic[] = [];
    const fragment = this.fragmentIndex.get(fragmentName);

    if (!fragment) {
      return diagnostics; // Fragment not found - different error
    }

    // Check required arguments are provided
    const providedArgs = new Set(
      directive.arguments?.map(a => a.name.value) ?? []
    );

    for (const arg of fragment.arguments) {
      if (arg.required && !providedArgs.has(arg.name)) {
        diagnostics.push({
          severity: DiagnosticSeverity.Error,
          range: this.getRange(directive),
          message: `Missing required argument '${arg.name}' of type ${arg.type}!`,
          source: 'houdini-lsp'
        });
      }
    }

    // Check for unknown arguments
    const validArgNames = new Set(fragment.arguments.map(a => a.name));
    for (const providedArg of directive.arguments ?? []) {
      if (!validArgNames.has(providedArg.name.value)) {
        diagnostics.push({
          severity: DiagnosticSeverity.Warning,
          range: this.getRange(providedArg),
          message: `Unknown argument '${providedArg.name.value}' for fragment ${fragmentName}`,
          source: 'houdini-lsp'
        });
      }
    }

    // TODO: Type validation (check if value matches declared type)

    return diagnostics;
  }

  /**
   * Get range from a GraphQL AST node
   */
  private getRange(node: any): Range {
    return {
      start: {
        line: (node.loc?.startToken.line ?? 1) - 1, // LSP is 0-indexed
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
  private visitDocument(
    document: DocumentNode,
    callback: (spread: FragmentSpreadNode) => void
  ): void {
    // Simple visitor - in production use graphql/language/visitor
    const visit = (node: any) => {
      if (node.kind === 'FragmentSpread') {
        callback(node);
      }

      // Recursively visit children
      for (const key in node) {
        const value = node[key];
        if (Array.isArray(value)) {
          value.forEach(visit);
        } else if (value && typeof value === 'object') {
          visit(value);
        }
      }
    };

    document.definitions.forEach(visit);
  }
}
