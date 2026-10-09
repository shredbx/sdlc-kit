import type { Schema } from './repository';

export interface DataModule<TRecord> {
  schema: Schema<TRecord>;
  repository: {
    findById(id: string): Promise<TRecord | null>;
    findMany(): Promise<readonly TRecord[]>;
  };
}

export interface DataRegistry<TModules> {
  get<TKey extends keyof TModules>(key: TKey): TModules[TKey];
}

export function createDataRegistry<TModules extends Record<string, unknown>>(
  modules: TModules
): DataRegistry<TModules> {
  const registeredModules = Object.freeze({ ...modules });

  return Object.freeze({
    get<TKey extends keyof TModules>(key: TKey): TModules[TKey] {
      return registeredModules[key];
    }
  });
}
