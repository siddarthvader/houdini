export interface Location {
  line: number;
  column: number;
}

/**
 * Represents a parsed fragment argument from @arguments directive
 */
export interface FragmentArgument {
  name: string;           // e.g., "width"
  type: string;           // e.g., "Int"
  required: boolean;      // true if type ends with !
  default?: any;          // default value if specified
  location: Location;     // for error reporting
}

/**
 * Represents a fragment with its arguments
 */
export interface FragmentDefinition {
  name: string;               // Fragment name
  typeCondition: string;      // e.g., "User"
  arguments: FragmentArgument[];
  uri: string;                // Document URI
  location: Location;
}

/**
 * Index of all fragments in workspace
 */
export interface FragmentIndex {
  fragments: Map<string, FragmentDefinition>;

  set(fragment: FragmentDefinition): void;
  get(name: string): FragmentDefinition | undefined;
  removeByUri(uri: string): void;
  getAll(): FragmentDefinition[];
}

