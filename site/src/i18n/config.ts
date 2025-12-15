import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import zh from './locales/zh.json';
import en from './locales/en.json';

const resources = {
  zh: { translation: zh },
  en: { translation: en },
};

// 从 localStorage 获取保存的语言设置，默认为中文
const savedLanguage = localStorage.getItem('vstats_language') || 'zh';

i18n
  .use(initReactI18next)
  .init({
    resources,
    lng: savedLanguage,
    fallbackLng: 'zh',
    interpolation: {
      escapeValue: false, // React 已经转义了
    },
  });

export default i18n;
