import { useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { motion } from 'framer-motion';
import { Activity, Server, Shield, Zap, Globe, Cpu } from 'lucide-react';
import { useTranslation } from 'react-i18next';

const fadeIn = {
  initial: { opacity: 0, y: 20 },
  whileInView: { opacity: 1, y: 0 },
  viewport: { once: true },
  transition: { duration: 0.5 }
};

export default function Home() {
  const { t } = useTranslation();
  const location = useLocation();
  const [installMethod, setInstallMethod] = useState<'docker' | 'manual'>('docker');

  useEffect(() => {
    // 处理 URL 中的 hash（如 /#features）
    const handleHashScroll = () => {
      const hash = window.location.hash;
      if (hash) {
        const elementId = hash.substring(1); // 去掉 # 号
        // 等待 DOM 渲染完成
        setTimeout(() => {
          const element = document.getElementById(elementId);
          if (element) {
            const offset = 80; // 导航栏高度
            const elementPosition = element.getBoundingClientRect().top + window.pageYOffset;
            const offsetPosition = elementPosition - offset;
            
            window.scrollTo({
              top: offsetPosition,
              behavior: 'smooth'
            });
          }
        }, 150);
      }
    };

    // 检查初始 hash
    handleHashScroll();

    // 监听 hash 变化
    window.addEventListener('hashchange', handleHashScroll);
    
    return () => {
      window.removeEventListener('hashchange', handleHashScroll);
    };
  }, [location.pathname]);
  return (
    <div className="overflow-hidden">
      {/* Hero Section */}
      <section className="relative min-h-screen flex items-center pt-20">
        <div className="absolute inset-0 pointer-events-none">
          <div className="absolute top-[-10%] left-[-10%] w-[500px] h-[500px] bg-sky-500/20 rounded-full blur-[100px] animate-blob" />
          <div className="absolute bottom-[-10%] right-[-10%] w-[500px] h-[500px] bg-emerald-500/20 rounded-full blur-[100px] animate-blob animation-delay-2000" />
        </div>
        
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 w-full">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center">
            <motion.div 
              initial={{ opacity: 0, x: -50 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.8 }}
            >
              <h1 className="text-5xl md:text-7xl font-bold tracking-tight mb-6 bg-clip-text text-transparent bg-gradient-to-r from-sky-500 to-emerald-500">
                {t('home.title')}
              </h1>
              <p className="text-2xl md:text-3xl font-bold text-slate-800 dark:text-slate-100 mb-6">
                {t('home.subtitle')}
              </p>
              <p className="text-lg text-slate-600 dark:text-slate-400 mb-8 leading-relaxed">
                {t('home.description')}
              </p>
              <div className="flex flex-wrap gap-4">
                <a href="#install" className="btn btn-primary">
                  {t('home.getStarted')}
                </a>
                <a href="https://github.com/zsai001/vstats" target="_blank" rel="noreferrer" className="btn btn-outline">
                  {t('common.github')}
                </a>
              </div>
              
              <div className="mt-12 flex gap-8">
                <div>
                  <div className="text-3xl font-bold text-sky-500">&lt;50ms</div>
                  <div className="text-sm text-slate-500">{t('home.dataLatency')}</div>
                </div>
                <div>
                  <div className="text-3xl font-bold text-emerald-500">1-Click</div>
                  <div className="text-sm text-slate-500">{t('home.oneClickDeploy')}</div>
                </div>
                <div>
                  <div className="text-3xl font-bold text-violet-500">100%</div>
                  <div className="text-sm text-slate-500">{t('home.openSource')}</div>
                </div>
              </div>
            </motion.div>
            
            <motion.div
              initial={{ opacity: 0, scale: 0.8 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ duration: 0.8, delay: 0.2 }}
              className="relative"
            >
              <div className="glass-card rounded-2xl p-6 relative z-10">
                <div className="flex items-center gap-2 mb-6">
                  <div className="w-3 h-3 rounded-full bg-red-500" />
                  <div className="w-3 h-3 rounded-full bg-yellow-500" />
                  <div className="w-3 h-3 rounded-full bg-green-500" />
                </div>
                
                <div className="space-y-6">
                  {['CPU Usage', 'Memory', 'Disk I/O', 'Network'].map((metric, i) => (
                    <div key={metric} className="space-y-2">
                      <div className="flex justify-between text-sm font-medium dark:text-slate-300">
                        <span>{metric}</span>
                        <span>{65 + i * 5}%</span>
                      </div>
                      <div className="h-2 bg-slate-200 dark:bg-slate-700 rounded-full overflow-hidden">
                        <motion.div 
                          initial={{ width: 0 }}
                          animate={{ width: `${65 + i * 5}%` }}
                          transition={{ duration: 1, delay: 0.5 + i * 0.1 }}
                          className={`h-full rounded-full bg-gradient-to-r ${
                            i % 2 === 0 ? 'from-sky-500 to-blue-500' : 'from-emerald-500 to-teal-500'
                          }`} 
                        />
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </motion.div>
          </div>
        </div>
      </section>

      {/* Features */}
      <section id="features" className="py-24 bg-white/50 dark:bg-slate-900/50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <motion.div {...fadeIn} className="text-center mb-16">
            <h2 className="text-3xl font-bold mb-4 dark:text-white">{t('home.features.title')}</h2>
            <p className="text-slate-500 max-w-2xl mx-auto">
              {t('home.features.subtitle')}
            </p>
          </motion.div>
          
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
            <FeatureCard 
              icon={Zap} 
              title={t('home.features.realtime.title')} 
              desc={t('home.features.realtime.desc')} 
            />
            <FeatureCard 
              icon={Server} 
              title={t('home.features.multiServer.title')} 
              desc={t('home.features.multiServer.desc')} 
            />
            <FeatureCard 
              icon={Activity} 
              title={t('home.features.modernUI.title')} 
              desc={t('home.features.modernUI.desc')} 
            />
            <FeatureCard 
              icon={Globe} 
              title={t('home.features.oneClick.title')} 
              desc={t('home.features.oneClick.desc')} 
            />
            <FeatureCard 
              icon={Shield} 
              title={t('home.features.security.title')} 
              desc={t('home.features.security.desc')} 
            />
            <FeatureCard 
              icon={Cpu} 
              title={t('home.features.comprehensive.title')} 
              desc={t('home.features.comprehensive.desc')} 
            />
          </div>
        </div>
      </section>

      {/* Theme Showcase */}
      <section className="py-24 bg-white/50 dark:bg-slate-900/50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <motion.div {...fadeIn} className="text-center mb-16">
            <h2 className="text-3xl font-bold mb-4 dark:text-white">{t('home.themes.title')}</h2>
            <p className="text-slate-500 max-w-2xl mx-auto">
              {t('home.themes.subtitle')}
            </p>
          </motion.div>
          
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
            {[1, 2, 3, 4, 5, 6].map((i) => (
              <motion.div 
                key={i}
                {...fadeIn}
                className="rounded-xl overflow-hidden shadow-xl border border-slate-200 dark:border-slate-800 group"
              >
                <div className="aspect-video relative overflow-hidden bg-slate-100 dark:bg-slate-800">
                  <img 
                    src={`/theme/${i}.png`} 
                    alt={`Theme ${i}`}
                    className="w-full h-full object-cover transform group-hover:scale-105 transition-transform duration-500"
                  />
                  <div className="absolute inset-0 bg-black/0 group-hover:bg-black/10 transition-colors duration-300" />
                </div>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Installation */}
      <section id="install" className="py-24 bg-slate-50 dark:bg-slate-800/50">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
          <motion.div {...fadeIn} className="text-center mb-16">
            <h2 className="text-3xl font-bold mb-4 dark:text-white">{t('home.installation.title')}</h2>
            <p className="text-slate-500">{t('home.installation.subtitle')}</p>
          </motion.div>

          <div className="glass-card rounded-xl overflow-hidden">
            <div className="flex border-b border-slate-200 dark:border-slate-700 bg-slate-100/50 dark:bg-slate-800/50">
              <button
                onClick={() => setInstallMethod('docker')}
                className={`px-6 py-3 font-medium transition-colors ${
                  installMethod === 'docker'
                    ? 'border-b-2 border-sky-500 text-sky-500'
                    : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
                }`}
              >
                {t('home.installation.docker')}
              </button>
              <button
                onClick={() => setInstallMethod('manual')}
                className={`px-6 py-3 font-medium transition-colors ${
                  installMethod === 'manual'
                    ? 'border-b-2 border-sky-500 text-sky-500'
                    : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
                }`}
              >
                {t('home.installation.manual')}
              </button>
            </div>
            <div className="p-6 overflow-x-auto">
              {installMethod === 'docker' ? (
                <pre className="font-mono text-sm text-slate-700 dark:text-slate-300 whitespace-pre">
{`# Create data directory
mkdir -p data && sudo chown -R 1000:1000 data

# Run container
docker run -d \\
  --name vstats-server \\
  -p 3001:3001 \\
  -v $(pwd)/data:/app/data \\
  --restart unless-stopped \\
  zsai001/vstats-server:latest`}
                </pre>
              ) : (
                <pre className="font-mono text-sm text-slate-700 dark:text-slate-300 whitespace-pre">
{`# One-click installation script
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash

# Or using wget
wget -qO- https://vstats.zsoft.cc/install.sh | sudo bash

# After installation, the service will be available at:
# http://your-server-ip:3001`}
                </pre>
              )}
            </div>
          </div>
        </div>
      </section>

      {/* Tech Stack */}
      <section className="py-24">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <motion.div {...fadeIn} className="text-center mb-16">
            <h2 className="text-3xl font-bold mb-4 dark:text-white">{t('home.techStack.title')}</h2>
            <p className="text-slate-500">{t('home.techStack.subtitle')}</p>
          </motion.div>
          
          <div className="flex flex-wrap justify-center gap-8 md:gap-12 grayscale hover:grayscale-0 transition-all duration-500">
            {[
              { name: 'Go', icon: 'https://cdn.jsdelivr.net/gh/devicons/devicon@latest/icons/go/go-original.svg' },
              { name: 'React', icon: 'https://cdn.jsdelivr.net/gh/devicons/devicon@latest/icons/react/react-original.svg' },
              { name: 'TypeScript', icon: 'https://cdn.jsdelivr.net/gh/devicons/devicon@latest/icons/typescript/typescript-original.svg' },
              { name: 'Tailwind', icon: 'https://cdn.jsdelivr.net/gh/devicons/devicon@latest/icons/tailwindcss/tailwindcss-original.svg' },
              { name: 'Docker', icon: 'https://cdn.jsdelivr.net/gh/devicons/devicon@latest/icons/docker/docker-original.svg' },
              { name: 'SQLite', icon: 'https://cdn.jsdelivr.net/gh/devicons/devicon@latest/icons/sqlite/sqlite-original.svg' }
            ].map((tech) => (
              <div key={tech.name} className="flex flex-col items-center gap-3 group">
                <img src={tech.icon} alt={tech.name} className="h-12 w-auto group-hover:-translate-y-2 transition-transform duration-300" />
                <span className="font-semibold text-slate-400 group-hover:text-slate-600 dark:group-hover:text-slate-200">{tech.name}</span>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Sponsors */}
      <section className="py-24 bg-slate-50 dark:bg-slate-800/50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <motion.div {...fadeIn} className="text-center mb-16">
            <h2 className="text-3xl font-bold mb-4 dark:text-white">{t('home.sponsors.title')}</h2>
            <p className="text-slate-500">{t('home.sponsors.subtitle')}</p>
          </motion.div>
          
          <div className="flex flex-wrap justify-center gap-8">
            <a href="https://www.tohu.cloud" target="_blank" rel="noreferrer" className="glass-card px-12 py-6 rounded-xl hover:-translate-y-1 transition-transform flex items-center">
              <span className="text-lg font-semibold text-slate-700 dark:text-slate-300">TOHU Cloud</span>
            </a>
            <a href="https://debee.io/" target="_blank" rel="noreferrer" className="glass-card px-12 py-6 rounded-xl hover:-translate-y-1 transition-transform flex items-center">
              <span className="text-lg font-semibold text-slate-700 dark:text-slate-300">Debee</span>
            </a>
          </div>
        </div>
      </section>
    </div>
  );
}

function FeatureCard({ icon: Icon, title, desc }: { icon: any, title: string, desc: string }) {
  return (
    <motion.div 
      {...fadeIn}
      className="glass-card p-8 rounded-2xl hover:-translate-y-1 transition-transform"
    >
      <div className="w-12 h-12 bg-sky-100 dark:bg-sky-900/30 rounded-xl flex items-center justify-center mb-6 text-sky-500">
        <Icon className="w-6 h-6" />
      </div>
      <h3 className="text-xl font-bold mb-3 dark:text-white">{title}</h3>
      <p className="text-slate-500 dark:text-slate-400 leading-relaxed">
        {desc}
      </p>
    </motion.div>
  );
}
