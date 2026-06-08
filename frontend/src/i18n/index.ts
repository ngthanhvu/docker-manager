import en from './en';

type MessageTree = Record<string, unknown>;
type TranslateParams = Record<string, string | number | boolean | null | undefined>;

const getMessage = (key: string) => {
  const value = key.split('.').reduce<unknown>((current, part) => {
    if (!current || typeof current !== 'object') return undefined;
    return (current as MessageTree)[part];
  }, en);

  return typeof value === 'string' ? value : key;
};

export const t = (key: string, params: TranslateParams = {}) =>
  getMessage(key).replace(/\{(\w+)\}/g, (_, name: string) => String(params[name] ?? `{${name}}`));

export const useI18n = () => ({ t });

export const i18n = {
  global: {
    t,
  },
};
