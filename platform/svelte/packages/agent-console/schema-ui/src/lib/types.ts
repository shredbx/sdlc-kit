/** Generic JSON Schema shape (the same wire format agent-framework's introspection routes
 * return, and the same one PydanticAI generates for tool-calling) - not tied to any one
 * consumer's response types, just JSON Schema's own vocabulary. */
export interface JsonSchema {
	type?: string;
	enum?: string[];
	anyOf?: JsonSchema[];
	description?: string;
	format?: string;
	properties?: Record<string, JsonSchema>;
	required?: string[];
	items?: JsonSchema;
	/** Pydantic's model_json_schema() ref format: "#/$defs/PropertyResult". */
	$ref?: string;
	/** Only present on a root schema (e.g. a tool's whole output_json_schema) - named nested
	 * model schemas that $ref points into. */
	$defs?: Record<string, JsonSchema>;
	title?: string;
}

/** Minimal shape SchemaForm needs - a real ToolSchema (with extra fields like `kind`) satisfies
 * this structurally, no explicit coupling needed. */
export interface ToolLike {
	name: string;
	description: string | null;
	parameters_json_schema: JsonSchema;
}

/** Matches agent_framework.core.cards.Card - what an AgentReply's cards field carries. */
export interface CardData {
	id: string;
	title: string;
	subtitle?: string | null;
	price_display?: string | null;
	image_url?: string | null;
	link?: string | null;
}

export interface RecordFieldSpec {
	label: string;
	value: string;
	rows?: number;
}
