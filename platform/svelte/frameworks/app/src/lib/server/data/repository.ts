export interface Schema<TRecord> {
  parse(value: unknown): TRecord;
}

export interface ReadRepository<TRecord> {
  findById(id: string): Promise<TRecord | null>;
  findMany(): Promise<readonly TRecord[]>;
}

export function createReadOnlyRepository<TRecord>(options: {
  records: readonly unknown[];
  schema: Schema<TRecord>;
  getId: (record: TRecord) => string;
}): ReadRepository<TRecord> {
  return {
    async findById(id) {
      for (const record of options.records) {
        const parsed = options.schema.parse(record);
        if (options.getId(parsed) === id) {
          return parsed;
        }
      }

      return null;
    },
    async findMany() {
      return options.records.map((record) => options.schema.parse(record));
    }
  };
}
