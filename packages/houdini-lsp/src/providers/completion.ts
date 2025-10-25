import {
  CompletionItem,
  CompletionItemKind,
  TextDocumentPositionParams,
  CompletionList,
  Connection
} from 'vscode-languageserver';
import { FragmentIndex, FragmentArgument } from '../types';

export class CompletionProvider {
  constructor(
    private fragmentIndex: FragmentIndex,
    private connection?: Connection
  ) {}

  /**
   * Provide completions for @with directive arguments
   */
  public provideCompletions(
    params: TextDocumentPositionParams,
    documentText: string
  ): CompletionList | null {
    const { position } = params;

    // Get the line of text up to cursor position
    const lines = documentText.split('\n');
    const line = lines[position.line];
    const textBeforeCursor = line.substring(0, position.character);

    this.connection?.console.error(`[Completion] Text before cursor: "${textBeforeCursor}"`);

    // Check if we're inside @with(...) directive
    const fragmentName = this.getFragmentNameForWith(textBeforeCursor, documentText);
    if (!fragmentName) {
      this.connection?.console.error('[Completion] No fragment name found');
      return null;
    }

    this.connection?.console.error(`[Completion] Fragment: ${fragmentName}`);

    const allFragments = this.fragmentIndex.getAll();
    this.connection?.console.error(`[Completion] Available fragments: ${allFragments.map(f => f.name).join(', ')}`);
    // Lookup fragment definition
    const fragment = this.fragmentIndex.get(fragmentName);
    if (!fragment) {
      this.connection?.console.error(`[Completion] Fragment "${fragmentName}" not in index`);
      this.connection?.console.error(`[Completion] Available fragments: ${Array.from(this.fragmentIndex.fragments.keys()).join(', ')}`);
      return null;
    }

    if (fragment.arguments.length === 0) {
      this.connection?.console.error(`[Completion] Fragment "${fragmentName}" has no arguments`);
      return null;
    }

    this.connection?.console.error(`[Completion] Found ${fragment.arguments.length} arguments: ${fragment.arguments.map(a => a.name).join(', ')}`);

    // Generate completion items for each argument
    const items = fragment.arguments.map(arg =>
      this.createCompletionItem(arg)
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
  private getFragmentNameForWith(
    textBeforeCursor: string,
    fullText: string
  ): string | null {
    // Match pattern: ...FragmentName @with(
    const match = textBeforeCursor.match(/\.\.\.(\w+)\s+@with\s*\(/);
    if (match) {
      return match[1];
    }

    // TODO: More robust parsing using GraphQL AST
    return null;
  }

  /**
   * Create completion item for a fragment argument
   */
  private createCompletionItem(arg: FragmentArgument): CompletionItem {
    const typeDisplay = arg.required ? `${arg.type}!` : arg.type;

    let documentation = `Type: ${typeDisplay}`;
    if (arg.default !== undefined) {
      documentation += `\nDefault: ${JSON.stringify(arg.default)}`;
    }
    if (arg.required) {
      documentation += '\n⚠️ Required argument';
    }

    return {
      label: arg.name,
      kind: CompletionItemKind.Property,
      detail: typeDisplay,
      documentation: {
        kind: 'markdown',
        value: documentation
      },
      insertText: `${arg.name}: `,
      sortText: arg.required ? `0_${arg.name}` : `1_${arg.name}` // Required args first
    };
  }
}

