// @sbx/core-ui types — shared type definitions for SBX workspace entities
// Extracted from sbx app's api.ts for use across all workspace apps
//
// These types represent the API contract between Go backend and Svelte frontends.
// They are consumed by core UI components and can be imported by any app.

// =============================================================================
// ENTITY DETAIL — Universal entity structure for detail views
// =============================================================================

/** Universal entity detail — shared fields from entity protocol */
export interface EntityDetail {
	name: string;
	type: string;
	purpose: string;
	vision?: string;
	motivation?: {
		drivers?: string[];
		stakeholders?: { role: string; interest: string }[];
		constraints?: string[];
	};
	goals?: {
		id: string;
		text: string;
		milestone?: string;
		status?: string;
		assigned_to?: string[];
		scope?: string[];
		refs?: string[];
		decomposed_to?: { id: string; entity: string; text: string; capability?: string }[];
	}[];
	pipeline?: {
		id: string;
		name: string;
		order: number;
		status: 'completed' | 'in_progress' | 'pending';
		description: string;
	}[];
	conforms_to?: string[];
	owner?: { name: string; email: string };
	structure?: Record<string, string>;
	fdd_layers?: Record<string, { purpose: string; components?: string[]; technologies?: string[] }>;
	entry_point?: { command: string; source: string };
}

// =============================================================================
// PIPELINE & RUNTIME
// =============================================================================

export interface PipelineScope {
	name: string;
	pipelines: string[];
	path: string;
}

export interface Pipeline {
	type: string;
	name: string;
	scope: string;
	purpose: string;
	endless: boolean;
	steps: Record<string, PipelineStep>;
	coordinator?: PipelineCoordinator;
	path: string;
	modified: string;
}

export interface PipelineStep {
	path: string;
	order: number;
	suborder?: string;
	description?: string;
	depends_on?: string[];
}

export interface PipelineCoordinator {
	system_prompt?: string;
	agent_profile?: string;
	role?: string;
	personality?: string;
	guidelines?: string[];
	auto_execution?: {
		enabled: boolean;
		stop_on_questions: boolean;
		parallel_allowed: boolean;
		require_artifacts: boolean;
	};
}

export interface RuntimeInstance {
	name: string;
	scope: string;
	pipeline: string;
	path: string;
	created: string;
	status: string;
}

export interface Task {
	type: string;
	id: string;
	name: string;
	purpose: string;
	brief: string;
	status: string;
	task_type: string;
	agent_type: string;
	current_phase?: string;
	next_task?: string;
	business_problem?: string;
	phases?: TaskPhase[];
	tasks?: string[];
	governance?: TaskGovernance;
	tags?: string[];
	path: string;
	modified: string;
}

export interface TaskPhase {
	id: string;
	name: string;
	purpose: string;
	impl_step?: string;
	governance?: string;
	entry_criteria?: CriterionItem[];
	exit_criteria?: CriterionItem[];
	tasks?: TaskItem[];
}

export interface CriterionItem {
	criterion: string;
	status: string;
	verified?: string;
	artifact?: string;
}

export interface TaskItem {
	id: string;
	task: string;
	status: string;
	depends_on?: string[];
	output?: string;
	artifact?: string;
}

export interface TaskGovernance {
	capabilities: string[];
	nfr: string[];
	rules_applied: string[];
	fdd_layer: { primary: string; sublayer?: string };
	reasoning: string;
	standards?: string[];
	patterns?: string[];
}

export interface TaskContract {
	task: Task;
	role?: string;
	personality?: string;
	context?: string;
	guidelines?: string[];
	prompt: string;
}

export interface DomainExpert {
	type: string;
	name: string;
	subject: string;
	status: 'draft' | 'researching' | 'complete' | 'stale';
	created: string;
	updated?: string;
	description: string;
	agent_profile?: string;
	scope?: DomainExpertScope;
	sources: DomainExpertSource[];
	outputs?: DomainExpertOutputs;
	knowledge?: DomainExpertKnowledge;
	pipeline?: DomainExpertPipeline;
	spectral?: DomainExpertSpectral;
	tags?: string[];
	path: string;
	modified: string;
}

export interface DomainExpertScope {
	topics: string[];
	external_sources?: string[];
	internal_artifacts?: string[];
	integration_points?: string[];
	expert_type: 'tool' | 'system' | 'business' | 'data';
}

export interface DomainExpertSource {
	name: string;
	type: 'documentation' | 'codebase' | 'book' | 'article' | 'video' | 'api' | 'internal';
	location: string;
	trust: 'canonical' | 'authoritative' | 'community' | 'experimental' | 'internal';
	priority: 'tier-1' | 'tier-2' | 'tier-3';
	notes?: string;
}

export interface DomainExpertOutputs {
	study_document: string;
	artifacts?: { name: string; path: string; schema?: string }[];
}

export interface DomainExpertKnowledge {
	path: string;
	auto_index?: boolean;
	freshness_days?: number;
}

export interface DomainExpertPipeline {
	current?: string;
	history?: { pipeline: string; completed: string; output?: string }[];
}

export interface DomainExpertSpectral {
	capabilities: string[];
	deliverables?: string[];
	knowledge_artifacts: string[];
	fdd_layers: string[];
}

export interface DomainExpertListResponse {
	total: number;
	experts: DomainExpert[];
}

export interface DomainExpertStudyDocResponse {
	expert: string;
	content: string;
}

export interface ProjectInfo {
	name: string;
	purpose: string;
	entity_count: number;
	path: string;
}

export interface EntitySummary {
	name: string;
	purpose: string;
	type_category?: string;
	pattern: string;
	property_count: number;
	function_count: number;
	catalogue_count: number;
	value_object_count: number;
	dictionary_count: number;
	value_count?: number;
	path: string;
}

export interface DictValue {
	code: string;
	label: string;
	description?: string;
	sort_order: number;
	icon?: string;
	deprecated?: boolean;
}

// =============================================================================
// IMPLEMENTATIONS & PLATFORM TYPES (Decisions #0032, #0033, #0034)
// =============================================================================

export interface PlatformImplementation {
	packages: PackageImplementation[];
}

export interface PackageImplementation {
	name: string;
	backend?: string;
	purpose?: string;
	components?: string[];
}

export interface EntityEnvironment {
	fixtures?: boolean;
	auto_migrate?: boolean;
	read_replicas?: boolean;
}

export interface EntityMotivation {
	vision?: string;
	goals?: EntityGoal[];
	drivers?: string[];
	stakeholders?: { role: string; interest: string }[];
	constraints?: string[];
}

export interface EntitySchema {
	type: string;
	name: string;
	conforms_to?: string[];
	type_category?: string;
	purpose: string;
	definition?: string;
	governance?: string;
	goals?: EntityGoal[];
	motivation?: EntityMotivation;
	decomposition_level?: { level: string; has_primitives_only: boolean };
	classification?: { pattern: string; description?: string };
	properties?: EntityProperty[];
	functions?: EntityFunction[];
	values?: DictValue[];
	entities?: SubEntityRef[];
	catalogues?: CatalogueRef[];
	value_objects?: ValueObjectRef[];
	entry_point?: { type: string; path: string; command?: string };
	implementation?: {
		storage?: { table: string; provider: string; migrations?: string };
		adapters?: { name: string; type: string; layer: string; technology: string; purpose: string }[];
	};
	implementations?: Record<string, PlatformImplementation>;
	environments?: Record<string, EntityEnvironment>;
	x_spectral?: EntitySpectral;
	x_tags?: string[];
	resolved_catalogues?: Catalogue[];
	resolved_value_objects?: ValueObject[];
	resolved_dictionaries?: Dictionary[];
	path: string;
	project: string;
	modified: string;
}

export interface EntityGoal {
	id: string;
	text: string;
	status?: string;
	assigned_to?: string[];
	scope?: string[];
	refs?: string[];
}

export interface EntityProperty {
	name: string;
	type: string;
	required: boolean;
	purpose: string;
	goal_ref?: string;
	constraints?: Record<string, unknown>;
	default?: unknown;
	catalogue?: string;
	value_object?: string;
	dictionary?: string;
	items?: string;
	enum?: string[];
}

export interface EntityFunction {
	name: string;
	purpose: string;
	goal_ref?: string;
	input?: string[];
	output?: unknown;
	rules?: string[];
}

export interface SubEntityRef {
	name: string;
	purpose: string;
	location: string;
	assigned_goals?: string[];
}

export interface CatalogueRef {
	name: string;
	path: string;
	purpose: string;
}

export interface ValueObjectRef {
	name: string;
	path: string;
	purpose: string;
}

export interface EntitySpectral {
	capability: string;
	knowledge_artifact: string;
	nfr?: string[];
	fdd_layer: { primary: string; sublayer?: string };
}

export interface Catalogue {
	type: string;
	name: string;
	purpose: string;
	description?: string;
	metadata?: { category: string; region?: string[]; entity_ref?: string };
	options: CatalogueOption[];
	constraints?: { required: boolean; default?: unknown };
	x_tags?: string[];
	path: string;
}

export interface CatalogueOption {
	id: string;
	name: string;
	description?: string;
	category?: string;
	default?: boolean;
}

export interface ValueObject {
	type: string;
	name: string;
	purpose: string;
	description?: string;
	metadata?: { category: string; entity_ref?: string };
	properties: ValueObjectProperty[];
	x_tags?: string[];
	path: string;
}

export interface ValueObjectProperty {
	name: string;
	type: string;
	required: boolean;
	purpose: string;
	category?: string;
	default?: unknown;
	constraints?: Record<string, unknown>;
	enum?: string[];
	items?: string;
}

export interface DictionaryRef {
	name: string;
	path: string;
	purpose: string;
}

export interface Dictionary {
	type: string;
	name: string;
	purpose: string;
	version?: string;
	description?: string;
	default_language?: string;
	supported_languages?: string[];
	categories?: DictionaryCategory[];
	options: DictionaryOption[];
	constraints?: DictionaryConstraints;
	metadata?: DictionaryMetadata;
	x_tags?: string[];
	path: string;
}

export interface DictionaryCategory {
	id: string;
	name: string;
	sort_order?: number;
}

export interface DictionaryOption {
	id: string;
	localization: Record<string, DictionaryLocalizedText>;
	media?: DictionaryMedia;
	category?: string;
	sort_order?: number;
	deprecated?: boolean;
	metadata?: Record<string, unknown>;
}

export interface DictionaryLocalizedText {
	name: string;
	description?: string;
	short_description?: string;
}

export interface DictionaryMedia {
	icon?: string;
	image?: string;
	thumbnail?: string;
	video?: string;
	animation?: string;
	voice?: string;
}

export interface DictionaryConstraints {
	required?: boolean;
	allow_multiple?: boolean;
	default?: unknown;
}

export interface DictionaryMetadata {
	entity_ref?: string;
	region?: string[];
	domain?: string;
	source?: string;
}

export interface DictionaryListResponse {
	project: string;
	entity: string;
	total: number;
	dictionaries: Dictionary[];
}

export interface ProjectListResponse {
	total: number;
	projects: ProjectInfo[];
}

export interface EntityListResponse {
	project: string;
	total: number;
	entities: EntitySummary[];
}

export interface CreateEntityInput {
	name: string;
	purpose: string;
	pattern: 'aggregate' | 'entity' | 'value-object';
	conforms_to?: string[];
	properties?: Partial<EntityProperty>[];
}

export interface CreatePropertyInput {
	name: string;
	type: string;
	required: boolean;
	purpose: string;
	goal_ref?: string;
	catalogue?: string;
	value_object?: string;
	constraints?: Record<string, unknown>;
	default?: unknown;
}

export interface CodeLocation {
	type: 'workspace' | 'submodule';
	path: string;
	url?: string;
	branch?: string;
	submodule_path?: string;
}

export interface Client {
	name: string;
	type: string;
	purpose: string;
	status: string;
	company?: string;
	projects: ClientProject[];
	tech_stack?: Record<string, unknown>;
	contacts?: { name?: string; email?: string; role?: string }[];
	capabilities?: string[];
}

export interface ClientProject {
	name: string;
	project_type: 'internal' | 'external';
	purpose?: string;
	submodule_path?: string;
	code_location?: CodeLocation;
	tech_stack?: Record<string, unknown>;
	dev?: {
		command?: string;
		directory?: string;
		port?: number;
	};
	build?: {
		command?: string;
		directory?: string;
	};
	test?: {
		command?: string;
		directory?: string;
	};
	deploy?: {
		command?: string;
		directory?: string;
		stage?: { command?: string };
		prod?: { command?: string };
	};
}

export interface CreateClientInput {
	name: string;
	purpose: string;
	company?: string;
	contacts?: { name: string; email: string; role?: string }[];
}

export interface CreateProjectInput {
	name: string;
	purpose: string;
	client?: string;
	entry_point: {
		type: 'api' | 'web' | 'cli' | 'library';
		path?: string;
	};
	tech_stack?: {
		language?: string;
		framework?: string;
	};
}

export interface ProjectSummary {
	name: string;
	purpose: string;
	project_type: string;
	entity_count: number;
	service_count: number;
	goal_count: number;
	has_fdd_layers: boolean;
	linked_client?: string;
	technologies: string[] | null;
	status: string;
	path: string;
}

export interface Project {
	type: string;
	name: string;
	conforms_to?: string[];
	purpose: string;
	project_type?: string;
	motivation?: {
		vision?: string;
		goals?: { id: string; text: string; milestone?: string; status?: string; assigned_to?: string[]; scope?: string[]; refs?: string[] }[];
		drivers?: string[];
		stakeholders?: { role: string; interest: string }[];
		actors?: Record<string, { role: string; label: string; capabilities?: string[]; scope?: string[] }>;
		constraints?: string[];
	};
	goals?: { id: string; text: string; refs?: string[] }[];
	vision?: string;
	fdd_layers?: {
		ui?: { purpose: string; types?: { name: string; technology: string; audience: string }[] };
		si?: { purpose: string; adapters?: { name: string; technology: string; provides: string }[] };
		md?: { purpose: string; entities?: { name: string; pattern: string; purpose: string }[] };
		pd?: { purpose: string; rules?: { id: string; name: string; description: string }[] };
	};
	interface_type?: string;
	entry_point?: { type: string; path: string; build?: string };
	services?: { name: string; type: string; language: string; framework: string; purpose: string; path: string }[];
	linked_client?: { name: string; projects?: { name: string; relationship: string }[] };
	packages?: { name: string; category?: string; used_by?: string; goals?: string[] }[];
	deliverables?: { name: string; capability: string; artifact: string; path: string; purpose: string }[];
	capabilities?: Record<string, { status: string; blocked_by?: string; gaps?: string[] }>;
	x_spectral?: { capability: string; knowledge_artifact: string; fdd_layer: { primary: string } };
	x_tags?: string[];
	path: string;
	modified: string;
	entity_count: number;
}

export interface ProjectSummaryListResponse {
	total: number;
	projects: ProjectSummary[];
}

export interface PackageSummary {
	name: string;
	purpose: string;
	category: string;
	version: string;
	language: string;
	framework?: string;
	export_count: number;
	path: string;
	capabilities?: string[];
}

export interface Package {
	type: string;
	name: string;
	version: string;
	purpose: string;
	package_category: string;
	language?: { primary: string; framework?: string; build_tool?: string };
	exports?: { name: string; type: string; description?: string; signature?: string }[];
	dependencies?: { name: string; type: string; version?: string }[];
	path: string;
	entry_file?: string;
	tests?: { path: string; command: string };
	capabilities?: string[];
}

export interface PackageListResponse {
	total: number;
	packages: PackageSummary[];
}

export interface ServiceSummary {
	name: string;
	purpose: string;
	version: string;
	language: string;
	framework?: string;
	module_count: number;
	endpoint_count: number;
	path: string;
	capabilities?: string[];
}

export interface Service {
	type: string;
	name: string;
	version: string;
	purpose: string;
	language?: { primary: string; framework?: string; build_tool?: string };
	modules?: { name: string; purpose: string; fdd_layer?: string }[];
	configuration?: { defaults?: Record<string, unknown> };
	lifecycle?: { events?: string[] };
	exports?: { name: string; type: string; description?: string; signature?: string }[];
	dependencies?: { name: string; type: string; version?: string }[];
	path: string;
	entry_file?: string;
	tests?: { path: string; command: string };
	capabilities?: string[];
	endpoints?: {
		type: string;
		name: string;
		purpose: string;
		path: string;
		method: string;
		service: string;
		operation: string;
	}[];
}

export interface ServiceListResponse {
	total: number;
	services: ServiceSummary[];
}

export interface PortAllocation {
	consumer: string;
	port: number;
	range: string;
	purpose: string;
	policy: string;
	allocated_at: string;
}

export interface RangeStatus {
	start: number;
	end: number;
	total: number;
	used: number;
	available: number;
	utilization: string;
}

export interface PortStatus {
	total_allocations: number;
	ranges: Record<string, RangeStatus>;
}

export interface InfraProject {
	name: string;
	purpose: string;
	has_docker_config: boolean;
	service_count: number;
	compose_file?: string;
	path: string;
}

export interface ServiceStatus {
	name: string;
	status: 'running' | 'stopped' | 'starting' | 'unhealthy' | 'unknown';
	port?: number;
	health?: string;
	container_id?: string;
	image?: string;
	uptime?: string;
}

export interface ProjectStatus {
	name: string;
	docker_enabled: boolean;
	docker_running: boolean;
	services: ServiceStatus[];
	total_services: number;
	running_count: number;
	stopped_count: number;
	unhealthy_count: number;
	error?: string;
}

export interface OperationResult {
	success: boolean;
	message: string;
	error?: string;
}

export interface PortListResponse {
	total: number;
	allocations: PortAllocation[];
}

export interface InfraProjectListResponse {
	total: number;
	projects: InfraProject[];
}

export interface EnvironmentStatus {
	name: string;
	running: boolean;
	running_services: number;
	total_services: number;
	error?: string;
}

export interface StepValidationSummary {
	name: string;
	order: number;
	description?: string;
	entry_valid: boolean;
	exit_valid: boolean;
	status: 'completed' | 'in_progress' | 'blocked' | 'pending';
	entry_satisfied: number;
	entry_total: number;
	exit_satisfied: number;
	exit_total: number;
}

export interface PipelineValidationSummary {
	system_path: string;
	pipeline_name: string;
	overall_valid: boolean;
	completed_steps: number;
	total_steps: number;
	next_step?: string;
	steps: StepValidationSummary[];
}

// =============================================================================
// DOCUMENT UI — Schema-driven components for document entity instances
// =============================================================================

export interface DocumentRecord {
	id: string;
	[key: string]: unknown;
	created_at?: string;
	updated_at?: string;
}

export interface DocumentDisplayConfig {
	title_attribute: string;
	subtitle_attribute?: string;
	image_attribute?: string;
	description_attribute?: string;
	badges?: Array<{ attribute: string; entries?: DictValue[] }>;
}

export interface AttributeSchema {
	name: string;
	type: string;
	purpose: string;
	required: boolean;
	multiplicity?: string;
	constraints?: Record<string, unknown>;
}

export interface SystemListItem {
	name: string;
	type: 'workspace' | 'project' | 'entity' | 'active-object';
	path: string;
}

export interface SystemListResponse {
	total: number;
	systems: SystemListItem[];
}

// =============================================================================
// COMPOUND VALUE TYPES — Money, Image Reference (Decision #0037)
// =============================================================================

/** Money value — amount in smallest currency unit + ISO 4217 3-letter uppercase code + fraction (decimal places) */
export interface MoneyValue {
	amount: number | null;
	currency: string;
	fraction: number;
}

/** Image document reference — UUID FK + cached CDN URL */
export interface ImageReference {
	id: string;
	url: string;
	sort_order?: number;
}

// =============================================================================
// ATTRIBUTE-GROUP TYPES — Dependent value groups (cascade-delete with parent)
// =============================================================================

/** Property location — geographic attributes (attribute-group → dependent table) */
export interface PropertyLocation {
	street: string | null;
	city: string | null;
	province: string | null;
	postal_code: string | null;
	region: string | null;
	lat: number | null;
	lng: number | null;
}

/** Property sizes — physical dimensions in sqm (attribute-group → dependent table) */
export interface PropertySizes {
	land_size: number | null;
	house_size: number | null;
	living_size: number | null;
}

/** Property rooms — room counts (attribute-group → dependent table) */
export interface PropertyRooms {
	bedrooms: number;
	bathrooms: number;
	kitchens: number;
	living_rooms: number;
}

/** Property features — boolean flags (attribute-group → dependent table) */
export interface PropertyFeatures {
	pool: boolean;
	garden: boolean;
	parking: boolean;
	security: boolean;
	furnished: boolean;
}

// =============================================================================
// PROPERTY DOCUMENT TYPES — Full CRUD types for property entity
// =============================================================================

/** Property summary — minimal data for card/list rendering */
export interface PropertySummary {
	id: string;
	title: string;
	cover_image_url: string | null;
	property_type: string | null;
	transaction_type: string | null;
	price_amount: number | null;
	price_currency: string;
	land_size: number | null;
	city: string | null;
	province: string | null;
	bedrooms: number | null;
	bathrooms: number | null;
	is_published: boolean;
	created_at: string;
}

/** Property — full document with all nested attribute-groups */
export interface Property extends PropertySummary {
	title_deed: string | null;
	text: string | null;
	cover_image_id: string | null;
	location: PropertyLocation | null;
	sizes: PropertySizes | null;
	rooms: PropertyRooms | null;
	features: PropertyFeatures | null;
	images: ImageReference[];
	updated_at: string;
	deleted_at: string | null;
	version: number;
	created_by: string | null;
	updated_by: string | null;
}

/** Property input — create/update payload sent to API */
export interface PropertyInput {
	title: string;
	title_deed?: string | null;
	text?: string | null;
	property_type?: string | null;
	transaction_type?: string | null;
	price_amount?: number | null;
	price_currency?: string;
	land_size?: number | null;
	cover_image_id?: string | null;
	is_published?: boolean;
	location?: PropertyLocation | null;
	sizes?: PropertySizes | null;
	rooms?: PropertyRooms | null;
	features?: PropertyFeatures | null;
	image_ids?: string[];
}

/** Property filters — query parameters for list filtering */
export interface PropertyFilters {
	property_type?: string | null;
	transaction_type?: string | null;
	min_price?: number | null;
	max_price?: number | null;
	province?: string | null;
	bedrooms_min?: number | null;
	is_published?: boolean | null;
	search?: string | null;
}
