import en from "./en.json";
import id from "./id.json";

type TranslationValue = string | Record<string, unknown>;
type Translations = Record<string, TranslationValue>;

interface TranslationMap {
	[locale: string]: Translations;
}

const translations: TranslationMap = {
	en: en as Translations,
	id: id as Translations,
};

let currentLocale = "en";

// Initialize locale from localStorage or browser preference
(function initLocale() {
	const savedLocale = localStorage.getItem("locale");
	if (savedLocale && translations[savedLocale]) {
		currentLocale = savedLocale;
	} else {
		const browserLang = navigator.language.split("-")[0];
		if (translations[browserLang]) {
			currentLocale = browserLang;
		}
	}
})();

/**
 * Get a nested value from an object using dot notation.
 */
function getNestedValue(obj: Translations, path: string): string | undefined {
	return path.split(".").reduce<string | undefined>(
		(acc, part) => {
			if (acc && typeof acc === "object") {
				return (acc as Record<string, unknown>)[part] as string | undefined;
			}
			return undefined;
		},
		obj as unknown as string | undefined,
	);
}

/**
 * Interpolate parameters into a translation string.
 * Replaces {key} placeholders with provided values.
 */
function interpolate(
	str: string,
	params: Record<string, string | number>,
): string {
	return str.replace(/\{(\w+)\}/g, (_, key: string) => {
		return params[key] !== undefined ? String(params[key]) : `{${key}}`;
	});
}

/**
 * Translate a key to the current locale.
 * Falls back to English if the key is not found in the current locale.
 *
 * @example
 *   t("auth.login")             // "Sign In"
 *   t("validation.minLength", { min: 8 })  // "Must be at least 8 characters"
 */
export function t(
	key: string,
	params: Record<string, string | number> = {},
): string {
	const translation = getNestedValue(translations[currentLocale], key);

	if (!translation) {
		const fallback = getNestedValue(translations["en"], key);
		if (fallback) {
			return interpolate(fallback, params);
		}
		return key;
	}

	return interpolate(translation, params);
}

/**
 * Set the current locale.
 */
export function setLocale(locale: string): void {
	if (translations[locale]) {
		currentLocale = locale;
		localStorage.setItem("locale", locale);
		document.documentElement.setAttribute("lang", locale);
	}
}

/**
 * Get the current locale code.
 */
export function getLocale(): string {
	return currentLocale;
}

/**
 * Get all available locale codes.
 */
export function getAvailableLocales(): string[] {
	return Object.keys(translations);
}

/** All available locale codes (convenience export). */
export const locales = Object.keys(translations);
