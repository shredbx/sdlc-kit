/**
 * IFixture Interface - Entity-Linked Test Data
 *
 * Fixtures enable the BUILD-ONCE-REUSE pattern:
 * - Same fixture works for component previews AND e2e tests
 * - Fixtures are linked to active-object entities
 * - Changes propagate to all usages automatically
 *
 * @example
 * // Simple fixture
 * const userFixture: IFixture = {
 *   name: 'john-doe',
 *   entity: 'user',
 *   purpose: 'Active user with full profile for happy path testing',
 *   data: {
 *     id: { type: 'uuid', value: '123e4567-e89b-12d3-a456-426614174000' },
 *     name: { type: 'string', value: 'John Doe' },
 *     email: { type: 'string', value: 'john@example.com' }
 *   }
 * };
 *
 * @example
 * // Fixture with generator
 * const dynamicUserFixture: IFixture = {
 *   name: 'random-user',
 *   entity: 'user',
 *   purpose: 'Random user data for fuzz testing',
 *   generatorConfig: { locale: 'en', seed: 12345 },
 *   data: {
 *     id: { type: 'uuid', faker: 'string.uuid' },
 *     name: { type: 'string', faker: 'person.fullName' },
 *     email: { type: 'string', faker: 'internet.email' }
 *   }
 * };
 */

// =============================================================================
// FIXTURE FIELD TYPES
// =============================================================================

/**
 * Base field definition in a fixture.
 */
export interface IFixtureFieldBase {
	/** Data type of field */
	type: string;

	/** Description of field */
	description?: string;
}

/**
 * Field with static value.
 */
export interface IFixtureFieldStatic extends IFixtureFieldBase {
	/** Static value */
	value: unknown;
}

/**
 * Field with Faker generator.
 */
export interface IFixtureFieldFaker extends IFixtureFieldBase {
	/** Faker.js method path (e.g., 'person.fullName') */
	faker: string;

	/** Generator options */
	options?: Record<string, unknown>;
}

/**
 * Field with custom generator function.
 */
export interface IFixtureFieldGenerator extends IFixtureFieldBase {
	/** Generator function name or expression */
	generator: string;

	/** Generator arguments */
	args?: unknown[];
}

/**
 * Field referencing another fixture.
 */
export interface IFixtureFieldRef extends IFixtureFieldBase {
	/** Reference in format 'entity:fixture-name' */
	ref: string;

	/** Field path to extract from referenced fixture */
	path?: string;
}

/**
 * Field with value transform.
 */
export interface IFixtureFieldTransform extends IFixtureFieldBase {
	/** Source field or value */
	source: string | unknown;

	/** Transform function name */
	transform: string;
}

/**
 * Union of all fixture field types.
 */
export type IFixtureField =
	| IFixtureFieldStatic
	| IFixtureFieldFaker
	| IFixtureFieldGenerator
	| IFixtureFieldRef
	| IFixtureFieldTransform;

// =============================================================================
// FIXTURE VARIANT
// =============================================================================

/**
 * Fixture variant for different test scenarios.
 */
export interface IFixtureVariant {
	/** Base fixture to extend */
	extends?: string;

	/** Fields to override from base */
	overrides?: Record<string, Partial<IFixtureField> | unknown>;

	/** Description of variant's test purpose */
	description?: string;

	/** Tags for variant */
	tags?: string[];
}

// =============================================================================
// FIXTURE USAGE
// =============================================================================

/**
 * Preview usage configuration.
 */
export interface IFixturePreviewUsage {
	/** Components using this fixture */
	components?: string[];

	/** Transform to apply for preview */
	transform?: string;

	/** Props to merge with fixture data */
	additionalProps?: Record<string, unknown>;
}

/**
 * E2E usage configuration.
 */
export interface IFixtureE2EUsage {
	/** Test suites using this fixture */
	tests?: string[];

	/** Whether to seed database with fixture */
	seed?: boolean;

	/** Database collection/table to seed */
	collection?: string;

	/** Cleanup after test */
	cleanup?: boolean;
}

/**
 * Combined usage configuration.
 */
export interface IFixtureUsage {
	/** Preview-specific usage */
	preview?: IFixturePreviewUsage;

	/** E2E-specific usage */
	e2e?: IFixtureE2EUsage;
}

// =============================================================================
// FIXTURE RELATION
// =============================================================================

/**
 * Relation to another fixture for complex test scenarios.
 */
export interface IFixtureRelation {
	/** Related entity type */
	entity: string;

	/** Related fixture name */
	fixture: string;

	/** Field to populate with relation ID */
	field: string;

	/** Field path in related fixture (default: 'id') */
	foreignField?: string;

	/** Type of relation */
	type?: 'one-to-one' | 'one-to-many' | 'many-to-one';
}

// =============================================================================
// FIXTURE INTERFACE
// =============================================================================

/**
 * IFixture - Entity-linked test data definition.
 *
 * Fixtures belong to entities and provide test data for:
 * - Component previews in Hub's /components page
 * - E2E tests in automated test suites
 * - Development and debugging
 */
export interface IFixture<TData = Record<string, IFixtureField>> {
	/** Unique fixture name within entity */
	name: string;

	/** Entity this fixture belongs to */
	entity: string;

	/** Description of what this fixture tests */
	purpose: string;

	/** Categorization tags */
	tags?: string[];

	/** Fixture data fields */
	data: TData;

	/** Named variations */
	variants?: Record<string, IFixtureVariant>;

	/** Usage tracking */
	usage?: IFixtureUsage;

	/** Related fixtures */
	relations?: IFixtureRelation[];

	/** Generator configuration (Faker options) */
	generatorConfig?: {
		locale?: string;
		seed?: number;
		[key: string]: unknown;
	};

	/** Whether fixture represents valid entity state (default: true) */
	valid?: boolean;

	/** Expected validation errors for invalid fixtures */
	expectedErrors?: string[];
}

// =============================================================================
// RESOLVED FIXTURE
// =============================================================================

/**
 * Resolved fixture with generated values.
 */
export interface IResolvedFixture<T = Record<string, unknown>> {
	/** Source fixture name */
	name: string;

	/** Source entity */
	entity: string;

	/** Resolved data with actual values */
	data: T;

	/** Applied variant name (if any) */
	variant?: string;

	/** Resolved relations */
	relations?: Record<string, IResolvedFixture>;
}

// =============================================================================
// FIXTURE REGISTRY
// =============================================================================

/**
 * Registry for managing fixtures.
 */
export interface IFixtureRegistry {
	/** All registered fixtures by entity */
	fixtures: Map<string, Map<string, IFixture>>;

	/** Register a fixture */
	register(fixture: IFixture): void;

	/** Get fixture by entity and name */
	get(entity: string, name: string): IFixture | undefined;

	/** Get all fixtures for an entity */
	getByEntity(entity: string): IFixture[];

	/** Get fixtures by tag */
	getByTag(tag: string): IFixture[];

	/** Resolve fixture to actual values */
	resolve<T = Record<string, unknown>>(
		entity: string,
		name: string,
		variant?: string
	): Promise<IResolvedFixture<T>>;
}

/**
 * Fixture registry implementation.
 */
export class FixtureRegistry implements IFixtureRegistry {
	fixtures = new Map<string, Map<string, IFixture>>();

	register(fixture: IFixture): void {
		if (!this.fixtures.has(fixture.entity)) {
			this.fixtures.set(fixture.entity, new Map());
		}
		this.fixtures.get(fixture.entity)!.set(fixture.name, fixture);
	}

	get(entity: string, name: string): IFixture | undefined {
		return this.fixtures.get(entity)?.get(name);
	}

	getByEntity(entity: string): IFixture[] {
		const entityFixtures = this.fixtures.get(entity);
		return entityFixtures ? Array.from(entityFixtures.values()) : [];
	}

	getByTag(tag: string): IFixture[] {
		const results: IFixture[] = [];
		for (const entityMap of this.fixtures.values()) {
			for (const fixture of entityMap.values()) {
				if (fixture.tags?.includes(tag)) {
					results.push(fixture);
				}
			}
		}
		return results;
	}

	async resolve<T = Record<string, unknown>>(
		entity: string,
		name: string,
		variant?: string
	): Promise<IResolvedFixture<T>> {
		const fixture = this.get(entity, name);
		if (!fixture) {
			throw new Error(`Fixture not found: ${entity}:${name}`);
		}

		// Resolve base data
		let data = await this.resolveFields(fixture.data, fixture.generatorConfig);

		// Apply variant if specified
		if (variant && fixture.variants?.[variant]) {
			const variantDef = fixture.variants[variant];

			// Extend from another fixture if specified
			if (variantDef.extends) {
				const baseFixture = this.get(entity, variantDef.extends);
				if (baseFixture) {
					const baseData = await this.resolveFields(baseFixture.data, fixture.generatorConfig);
					data = { ...baseData, ...data };
				}
			}

			// Apply overrides
			if (variantDef.overrides) {
				for (const [key, value] of Object.entries(variantDef.overrides)) {
					if (typeof value === 'object' && value !== null && 'type' in value) {
						data[key] = await this.resolveField(value as IFixtureField, fixture.generatorConfig);
					} else {
						data[key] = value;
					}
				}
			}
		}

		// Resolve relations
		const relations: Record<string, IResolvedFixture> = {};
		if (fixture.relations) {
			for (const rel of fixture.relations) {
				const relatedFixture = await this.resolve(rel.entity, rel.fixture);
				relations[rel.field] = relatedFixture;

				// Populate the foreign key field
				const foreignField = rel.foreignField ?? 'id';
				if (foreignField in relatedFixture.data) {
					data[rel.field] = relatedFixture.data[foreignField];
				}
			}
		}

		return {
			name,
			entity,
			data: data as T,
			variant,
			relations: Object.keys(relations).length > 0 ? relations : undefined
		};
	}

	private async resolveFields(
		fields: Record<string, IFixtureField>,
		config?: IFixture['generatorConfig']
	): Promise<Record<string, unknown>> {
		const result: Record<string, unknown> = {};

		for (const [key, field] of Object.entries(fields)) {
			result[key] = await this.resolveField(field, config);
		}

		return result;
	}

	private async resolveField(
		field: IFixtureField,
		config?: IFixture['generatorConfig']
	): Promise<unknown> {
		// Static value
		if ('value' in field) {
			return field.value;
		}

		// Faker generator
		if ('faker' in field) {
			return this.generateFakerValue(field.faker, field.options, config);
		}

		// Custom generator
		if ('generator' in field) {
			return this.runGenerator(field.generator, field.args);
		}

		// Reference to another fixture
		if ('ref' in field) {
			const [entity, name] = field.ref.split(':');
			const resolved = await this.resolve(entity, name);
			if (field.path) {
				return this.getPath(resolved.data, field.path);
			}
			return resolved.data;
		}

		// Transform
		if ('transform' in field) {
			const source =
				typeof field.source === 'string' ? this.evaluateExpression(field.source) : field.source;

			return this.applyTransform(source, field.transform);
		}

		return undefined;
	}

	private generateFakerValue(
		method: string,
		options?: Record<string, unknown>,
		config?: IFixture['generatorConfig']
	): unknown {
		// In real implementation, use @faker-js/faker
		// This is a placeholder that returns sensible defaults
		const generators: Record<string, () => unknown> = {
			'string.uuid': () => crypto.randomUUID(),
			'person.fullName': () => 'John Doe',
			'person.firstName': () => 'John',
			'person.lastName': () => 'Doe',
			'internet.email': () => 'john@example.com',
			'internet.url': () => 'https://example.com',
			'lorem.sentence': () => 'Lorem ipsum dolor sit amet.',
			'lorem.paragraph': () =>
				'Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.',
			'number.int': () => Math.floor(Math.random() * 1000),
			'date.past': () => new Date(Date.now() - Math.random() * 365 * 24 * 60 * 60 * 1000),
			'date.future': () => new Date(Date.now() + Math.random() * 365 * 24 * 60 * 60 * 1000),
			'datatype.boolean': () => Math.random() > 0.5
		};

		const generator = generators[method];
		if (generator) {
			return generator();
		}

		console.warn(`Unknown faker method: ${method}`);
		return `[${method}]`;
	}

	private runGenerator(name: string, args?: unknown[]): unknown {
		// Custom generator implementation
		// In real implementation, would look up registered generators
		console.warn(`Custom generator not implemented: ${name}`);
		return `[${name}]`;
	}

	private getPath(obj: Record<string, unknown>, path: string): unknown {
		const parts = path.split('.');
		let current: unknown = obj;

		for (const part of parts) {
			if (current === null || current === undefined) {
				return undefined;
			}
			current = (current as Record<string, unknown>)[part];
		}

		return current;
	}

	private evaluateExpression(expr: string): unknown {
		// Safe expression evaluation
		// In real implementation, would use a sandboxed evaluator
		return expr;
	}

	private applyTransform(value: unknown, transform: string): unknown {
		const transforms: Record<string, (v: unknown) => unknown> = {
			toUpperCase: (v) => String(v).toUpperCase(),
			toLowerCase: (v) => String(v).toLowerCase(),
			trim: (v) => String(v).trim(),
			toNumber: (v) => Number(v),
			toString: (v) => String(v),
			toBoolean: (v) => Boolean(v),
			toArray: (v) => (Array.isArray(v) ? v : [v]),
			first: (v) => (Array.isArray(v) ? v[0] : v),
			last: (v) => (Array.isArray(v) ? v[v.length - 1] : v),
			length: (v) => (Array.isArray(v) ? v.length : String(v).length)
		};

		const fn = transforms[transform];
		if (fn) {
			return fn(value);
		}

		console.warn(`Unknown transform: ${transform}`);
		return value;
	}
}

// =============================================================================
// GLOBAL REGISTRY
// =============================================================================

/** Global fixture registry */
export const fixtureRegistry = new FixtureRegistry();

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

/**
 * Create a simple fixture with static values.
 */
export function createFixture<T extends Record<string, unknown>>(
	name: string,
	entity: string,
	purpose: string,
	data: { [K in keyof T]: T[K] extends unknown ? IFixtureField : IFixtureField }
): IFixture {
	return {
		name,
		entity,
		purpose,
		data
	};
}

/**
 * Create a fixture field with static value.
 */
export function staticField(type: string, value: unknown): IFixtureFieldStatic {
	return { type, value };
}

/**
 * Create a fixture field with Faker generator.
 */
export function fakerField(type: string, faker: string, options?: Record<string, unknown>): IFixtureFieldFaker {
	return { type, faker, options };
}

/**
 * Create a fixture field referencing another fixture.
 */
export function refField(type: string, ref: string, path?: string): IFixtureFieldRef {
	return { type, ref, path };
}

/**
 * Load fixture for component preview.
 */
export async function loadFixtureForPreview<T = Record<string, unknown>>(
	entity: string,
	name: string,
	variant?: string,
	transform?: string
): Promise<T> {
	const resolved = await fixtureRegistry.resolve<T>(entity, name, variant);

	if (transform) {
		// Apply JSONPath or transform
		// In real implementation, use jsonpath library
		return resolved.data;
	}

	return resolved.data;
}
