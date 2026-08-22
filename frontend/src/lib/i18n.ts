import i18n from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import { initReactI18next } from 'react-i18next';

// Deep merge function for translations
function deepMerge<T extends Record<string, unknown>>(target: T, ...sources: T[]): T {
  for (const source of sources) {
    for (const key in source) {
      if (source[key] && typeof source[key] === 'object' && !Array.isArray(source[key])) {
        if (!target[key] || typeof target[key] !== 'object' || Array.isArray(target[key])) {
          (target as Record<string, unknown>)[key] = {};
        }
        deepMerge(target[key] as Record<string, unknown>, source[key] as Record<string, unknown>);
      } else {
        (target as Record<string, unknown>)[key] = source[key];
      }
    }
  }
  return target;
}

type LocaleModule = {
  default: Record<string, unknown>;
};

function getModuleDefaultExport(module: unknown): Record<string, unknown> {
  if (module && typeof module === 'object' && 'default' in module) {
    return (module as LocaleModule).default;
  }
  return module as Record<string, unknown>;
}

const enModules = import.meta.glob('../locales/en/*.json', { eager: true }) as Record<string, unknown>;
const zhCNModules = import.meta.glob('../locales/zh-CN/*.json', { eager: true }) as Record<string, unknown>;

const enTranslation = deepMerge({}, ...Object.values(enModules).map(getModuleDefaultExport)) as Record<string, unknown>;
const zhTranslation = deepMerge({}, ...Object.values(zhCNModules).map(getModuleDefaultExport)) as Record<string, unknown>;

const resources = {
  en: {
    translation: enTranslation,
  },
  zh: {
    translation: zhTranslation,
  },
  'zh-CN': {
    translation: zhTranslation,
  },
};

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: 'en',
    debug: false,
    supportedLngs: ['en', 'zh', 'zh-CN'],

    interpolation: {
      escapeValue: false, // React 已经默认转义了
      format: (value, format, lng, options) => {
        if (format === 'currency') {
          return new Intl.NumberFormat(options?.locale || lng, {
            style: 'currency',
            currency: options?.currency || 'USD',
            currencyDisplay: 'narrowSymbol',
            minimumFractionDigits: options?.minimumFractionDigits,
            maximumFractionDigits: options?.maximumFractionDigits,
          }).format(value);
        }
        return value;
      },
    },

    detection: {
      order: ['localStorage', 'navigator', 'htmlTag'],
      caches: ['localStorage'],
      convertDetectedLanguage: (lng: string) => {
        const normalized = lng.toLowerCase();
        if (normalized === 'zh-cn' || normalized.startsWith('zh-')) {
          return 'zh';
        }
        return lng;
      },
    },
  });

export default i18n;
