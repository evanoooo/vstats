import { Link } from 'react-router-dom';
import { Github, MessageCircle } from 'lucide-react';
import { useTranslation } from 'react-i18next';

// X (Twitter) Icon
const XIcon = ({ className = "w-5 h-5" }: { className?: string }) => (
  <svg className={className} viewBox="0 0 24 24" fill="currentColor">
    <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"/>
  </svg>
);

export default function Footer() {
  const { t } = useTranslation();
  return (
    <footer className="bg-white dark:bg-slate-900 border-t border-slate-200 dark:border-slate-800 py-12">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
          <div className="col-span-1 md:col-span-2">
            <div className="flex items-center gap-2 mb-4">
              <div className="bg-gradient-to-br from-sky-500 to-emerald-500 w-8 h-8 rounded-lg flex items-center justify-center text-white font-bold">
                V
              </div>
              <span className="text-xl font-bold dark:text-white">{t('common.vStats')}</span>
            </div>
            <p className="text-slate-500 dark:text-slate-400 max-w-sm">
              {t('footer.description')}
            </p>
          </div>
          
          <div>
            <h3 className="font-semibold mb-4 dark:text-white">{t('footer.product.title')}</h3>
            <ul className="space-y-2">
              <li><a href="#" className="text-slate-500 dark:text-slate-400 hover:text-sky-500 transition-colors">{t('footer.product.features')}</a></li>
              <li><a href="#" className="text-slate-500 dark:text-slate-400 hover:text-sky-500 transition-colors">{t('footer.product.integrations')}</a></li>
              <li><a href="#" className="text-slate-500 dark:text-slate-400 hover:text-sky-500 transition-colors">{t('footer.product.pricing')}</a></li>
              <li><a href="#" className="text-slate-500 dark:text-slate-400 hover:text-sky-500 transition-colors">{t('footer.product.changelog')}</a></li>
            </ul>
          </div>
          
          <div>
            <h3 className="font-semibold mb-4 dark:text-white">{t('footer.resources.title')}</h3>
            <ul className="space-y-2">
              <li><Link to="/docs" className="text-slate-500 dark:text-slate-400 hover:text-sky-500 transition-colors">{t('footer.resources.documentation')}</Link></li>
              <li><a href="https://github.com/zsai001/vstats" target="_blank" rel="noreferrer" className="text-slate-500 dark:text-slate-400 hover:text-sky-500 transition-colors">{t('footer.resources.github')}</a></li>
              <li><a href="https://github.com/zsai001/vstats/issues" target="_blank" rel="noreferrer" className="text-slate-500 dark:text-slate-400 hover:text-sky-500 transition-colors">{t('footer.resources.issues')}</a></li>
              <li><a href="https://github.com/zsai001/vstats/blob/master/LICENSE" target="_blank" rel="noreferrer" className="text-slate-500 dark:text-slate-400 hover:text-sky-500 transition-colors">{t('footer.resources.license')}</a></li>
            </ul>
          </div>
        </div>
        
        <div className="mt-12 pt-8 border-t border-slate-200 dark:border-slate-800 flex flex-col md:flex-row justify-between items-center gap-4">
          <p className="text-slate-500 dark:text-slate-400 text-sm">
            {t('footer.copyright')}
          </p>
          <div className="flex gap-6">
            <a 
              href="https://x.com/zsai010" 
              target="_blank" 
              rel="noreferrer" 
              className="text-slate-400 hover:text-sky-500 transition-colors flex items-center gap-2"
              aria-label="X (Twitter)"
            >
              <XIcon className="w-5 h-5" />
              <span className="hidden sm:inline">X</span>
            </a>
            <a 
              href="https://github.com/zsai001/vstats" 
              target="_blank" 
              rel="noreferrer" 
              className="text-slate-400 hover:text-sky-500 transition-colors flex items-center gap-2"
              aria-label="GitHub"
            >
              <Github className="w-5 h-5" />
              <span className="hidden sm:inline">GitHub</span>
            </a>
            <a 
              href="https://discord.gg/TTNky7Z4QM" 
              target="_blank" 
              rel="noreferrer" 
              className="text-slate-400 hover:text-sky-500 transition-colors flex items-center gap-2"
              aria-label="Discord"
            >
              <MessageCircle className="w-5 h-5" />
              <span className="hidden sm:inline">TalksDev</span>
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}
