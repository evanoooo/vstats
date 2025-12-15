import { Terminal, Download, Command } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export default function CLI() {
  const { t } = useTranslation();
  return (
    <div className="pt-20 min-h-screen">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-20">
        <div className="text-center mb-16">
          <div className="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-emerald-100 dark:bg-emerald-900/30 text-emerald-500 mb-8">
            <Terminal className="w-10 h-10" />
          </div>
          <h1 className="text-5xl font-bold mb-6 dark:text-white">{t('cli.title')}</h1>
          <p className="text-xl text-slate-500 max-w-2xl mx-auto">
            {t('cli.subtitle')}
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-8 mb-20">
          <div className="glass-card p-8 rounded-2xl">
            <Command className="w-8 h-8 text-emerald-500 mb-4" />
            <h3 className="text-xl font-bold mb-2 dark:text-white">{t('cli.fullControl.title')}</h3>
            <p className="text-slate-500">{t('cli.fullControl.desc')}</p>
          </div>
          <div className="glass-card p-8 rounded-2xl">
            <Download className="w-8 h-8 text-emerald-500 mb-4" />
            <h3 className="text-xl font-bold mb-2 dark:text-white">{t('cli.easyInstall.title')}</h3>
            <p className="text-slate-500">{t('cli.easyInstall.desc')}</p>
          </div>
          <div className="glass-card p-8 rounded-2xl">
            <Terminal className="w-8 h-8 text-emerald-500 mb-4" />
            <h3 className="text-xl font-bold mb-2 dark:text-white">{t('cli.automation.title')}</h3>
            <p className="text-slate-500">{t('cli.automation.desc')}</p>
          </div>
        </div>

        <div className="max-w-3xl mx-auto glass-card rounded-xl overflow-hidden">
          <div className="flex items-center gap-2 px-4 py-3 bg-slate-900 border-b border-slate-800">
            <div className="w-3 h-3 rounded-full bg-red-500" />
            <div className="w-3 h-3 rounded-full bg-yellow-500" />
            <div className="w-3 h-3 rounded-full bg-green-500" />
          </div>
          <div className="p-6 bg-slate-950 font-mono text-sm">
            <div className="text-slate-400 mb-2">$ vstats server list</div>
            <div className="text-emerald-500 mb-4">
              NAME          STATUS    CPU     MEM     UPTIME<br/>
              web-prod-01   <span className="text-green-500">●</span> online   45%     62%     12d 4h<br/>
              db-master     <span className="text-green-500">●</span> online   28%     84%     45d 1h<br/>
              cache-01      <span className="text-red-500">●</span> offline  -       -       -
            </div>
            <div className="text-slate-400 mb-2">$ vstats agent install --server=web-prod-02</div>
            <div className="text-slate-300">
              Installing vStats agent...<br/>
              Downloading binary... OK<br/>
              Configuring service... OK<br/>
              Agent started successfully!
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
