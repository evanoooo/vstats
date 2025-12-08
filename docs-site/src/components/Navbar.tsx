import { useState, useEffect } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { Menu, Moon, Sun, ChevronDown, Monitor, Terminal, Cloud as CloudIcon, Globe, LogOut } from 'lucide-react';
import clsx from 'clsx';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../auth/AuthContext';

const products = [
  {
    nameKey: 'vStats',
    descriptionKey: 'serverMonitoring',
    path: '/',
    icon: Monitor,
    color: 'text-sky-500'
  },
  {
    nameKey: 'vStatsCLI',
    descriptionKey: 'commandLineTool',
    path: '/cli',
    icon: Terminal,
    color: 'text-emerald-500'
  },
  {
    nameKey: 'vStatsCloud',
    descriptionKey: 'cloudPlatform',
    path: '/cloud',
    icon: CloudIcon,
    color: 'text-violet-500'
  },
];

export default function Navbar() {
  const { t, i18n } = useTranslation();
  const [isScrolled, setIsScrolled] = useState(false);
  const [isDark, setIsDark] = useState(true);
  const [isProductMenuOpen, setIsProductMenuOpen] = useState(false);
  const [isLangMenuOpen, setIsLangMenuOpen] = useState(false);
  const location = useLocation();
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [isUserMenuOpen, setIsUserMenuOpen] = useState(false);

  const currentProduct = products.find(p => p.path === location.pathname) || products[0];
  const currentLang = i18n.language;

  const changeLanguage = (lng: string) => {
    i18n.changeLanguage(lng);
    localStorage.setItem('vstats_language', lng);
    setIsLangMenuOpen(false);
  };

  const handleFeaturesClick = (e: React.MouseEvent<HTMLAnchorElement>) => {
    e.preventDefault();
    if (location.pathname !== '/') {
      // 如果不在首页，先导航到首页并设置 hash
      navigate('/');
      setTimeout(() => {
        window.location.hash = '#features';
      }, 100);
    } else {
      // 如果已在首页，直接设置 hash（Home 组件会处理滚动）
      window.location.hash = '#features';
      const featuresElement = document.getElementById('features');
      if (featuresElement) {
        const offset = 80; // 导航栏高度
        const elementPosition = featuresElement.getBoundingClientRect().top + window.pageYOffset;
        const offsetPosition = elementPosition - offset;
        window.scrollTo({
          top: offsetPosition,
          behavior: 'smooth'
        });
      }
    }
  };

  useEffect(() => {
    const handleScroll = () => setIsScrolled(window.scrollY > 20);
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  useEffect(() => {
    if (isDark) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }, [isDark]);

  return (
    <nav className={clsx(
      'fixed top-0 inset-x-0 z-50 transition-all duration-300',
      isScrolled ? 'bg-white/80 dark:bg-slate-900/80 backdrop-blur-md border-b border-slate-200 dark:border-slate-800' : 'bg-transparent'
    )}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Logo & Product Switcher */}
          <div className="relative">
            <button
              className="flex items-center gap-2 group focus:outline-none"
              onClick={() => setIsProductMenuOpen(!isProductMenuOpen)}
              onBlur={() => setTimeout(() => setIsProductMenuOpen(false), 200)}
            >
              <div className="bg-gradient-to-br from-sky-500 to-emerald-500 p-1.5 rounded-lg">
                <currentProduct.icon className="w-5 h-5 text-white" />
              </div>
              <div className="flex flex-col items-start">
                <span className="font-bold text-lg leading-none dark:text-white">
                  {t(`navbar.products.${currentProduct.nameKey}`)}
                </span>
              </div>
              <ChevronDown className={clsx(
                "w-4 h-4 text-slate-500 transition-transform duration-200",
                isProductMenuOpen && "rotate-180"
              )} />
            </button>

            <AnimatePresence>
              {isProductMenuOpen && (
                <motion.div
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: 10 }}
                  transition={{ duration: 0.2 }}
                  className="absolute top-full left-0 mt-2 w-64 bg-white dark:bg-slate-800 rounded-xl shadow-2xl border border-slate-200 dark:border-slate-700 overflow-hidden"
                >
                  <div className="p-2 space-y-1">
                    {products.map((product) => (
                      <Link
                        key={product.nameKey}
                        to={product.path}
                        className={clsx(
                          "flex items-center gap-3 p-3 rounded-lg transition-colors",
                          location.pathname === product.path
                            ? "bg-slate-100 dark:bg-slate-700/50"
                            : "hover:bg-slate-50 dark:hover:bg-slate-700/30"
                        )}
                        onClick={() => setIsProductMenuOpen(false)}
                      >
                        <div className={clsx("p-2 rounded-lg bg-slate-100 dark:bg-slate-900", product.color)}>
                          <product.icon className="w-5 h-5" />
                        </div>
                        <div>
                          <div className="font-medium dark:text-slate-200">{t(`navbar.products.${product.nameKey}`)}</div>
                          <div className="text-xs text-slate-500 dark:text-slate-400">{t(`navbar.products.${product.descriptionKey}`)}</div>
                        </div>
                      </Link>
                    ))}
                  </div>
                </motion.div>
              )}
            </AnimatePresence>
          </div>

          {/* Desktop Nav */}
          <div className="hidden md:flex items-center gap-8">
            <a
              href="#features"
              onClick={handleFeaturesClick}
              className="text-sm font-medium text-slate-600 dark:text-slate-300 hover:text-sky-500 transition-colors"
            >
              {t('navbar.features')}
            </a>
            <Link to="/docs" className="text-sm font-medium text-slate-600 dark:text-slate-300 hover:text-sky-500 transition-colors">{t('navbar.documentation')}</Link>
            <a href="https://github.com/zsai001/vstats" target="_blank" rel="noreferrer" className="text-sm font-medium text-slate-600 dark:text-slate-300 hover:text-sky-500 transition-colors">{t('navbar.github')}</a>

            <div className="flex items-center gap-3 pl-6 border-l border-slate-200 dark:border-slate-700">
              {/* Language Switcher */}
              <div className="relative">
                <button
                  onClick={() => setIsLangMenuOpen(!isLangMenuOpen)}
                  onBlur={() => setTimeout(() => setIsLangMenuOpen(false), 200)}
                  className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400 transition-colors flex items-center gap-1"
                >
                  <Globe className="w-5 h-5" />
                  <span className="text-xs font-medium">{currentLang.toUpperCase()}</span>
                </button>
                <AnimatePresence>
                  {isLangMenuOpen && (
                    <motion.div
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                      exit={{ opacity: 0, y: 10 }}
                      transition={{ duration: 0.2 }}
                      className="absolute top-full right-0 mt-2 w-32 bg-white dark:bg-slate-800 rounded-lg shadow-2xl border border-slate-200 dark:border-slate-700 overflow-hidden"
                    >
                      <button
                        onClick={() => changeLanguage('zh')}
                        className={clsx(
                          "w-full px-4 py-2 text-left text-sm transition-colors",
                          currentLang === 'zh'
                            ? "bg-sky-100 dark:bg-sky-900/30 text-sky-600 dark:text-sky-400"
                            : "hover:bg-slate-50 dark:hover:bg-slate-700/30 text-slate-600 dark:text-slate-400"
                        )}
                      >
                        中文
                      </button>
                      <button
                        onClick={() => changeLanguage('en')}
                        className={clsx(
                          "w-full px-4 py-2 text-left text-sm transition-colors",
                          currentLang === 'en'
                            ? "bg-sky-100 dark:bg-sky-900/30 text-sky-600 dark:text-sky-400"
                            : "hover:bg-slate-50 dark:hover:bg-slate-700/30 text-slate-600 dark:text-slate-400"
                        )}
                      >
                        English
                      </button>
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>
              <button
                onClick={() => setIsDark(!isDark)}
                className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400 transition-colors"
              >
                {isDark ? <Sun className="w-5 h-5" /> : <Moon className="w-5 h-5" />}
              </button>
              <a href="https://vps.zsoft.cc" className="btn-primary text-sm px-4 py-2 rounded-lg">
                {t('navbar.liveDemo')}
              </a>

              {/* User Menu - Only show when logged in */}
              {user && (
                <div className="relative pl-4 border-l border-slate-200 dark:border-slate-700">
                  <button
                    onClick={() => setIsUserMenuOpen(!isUserMenuOpen)}
                    onBlur={() => setTimeout(() => setIsUserMenuOpen(false), 200)}
                    className="flex items-center gap-2 hover:opacity-80 transition-opacity"
                  >
                    <div className="w-8 h-8 rounded-full bg-gradient-to-br from-violet-500 to-purple-600 flex items-center justify-center text-white font-medium shadow-sm">
                      {user.username.charAt(0).toUpperCase()}
                    </div>
                  </button>

                  <AnimatePresence>
                    {isUserMenuOpen && (
                      <motion.div
                        initial={{ opacity: 0, y: 10 }}
                        animate={{ opacity: 1, y: 0 }}
                        exit={{ opacity: 0, y: 10 }}
                        transition={{ duration: 0.2 }}
                        className="absolute top-full right-0 mt-2 w-48 bg-white dark:bg-slate-800 rounded-xl shadow-2xl border border-slate-200 dark:border-slate-700 overflow-hidden"
                      >
                        <div className="p-3 border-b border-slate-100 dark:border-slate-700/50">
                          <div className="text-sm font-medium text-slate-900 dark:text-slate-100">{user.username}</div>
                          <div className="text-xs text-slate-500 dark:text-slate-400 capitalize">{user.provider}</div>
                        </div>
                        <div className="p-1">
                          <Link
                            to="/cloud"
                            onClick={() => setIsUserMenuOpen(false)}
                            className="w-full flex items-center gap-2 px-3 py-2 text-sm text-slate-600 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700/50 rounded-lg transition-colors"
                          >
                            <CloudIcon className="w-4 h-4" />
                            {t('navbar.products.vStatsCloud', 'vStats Cloud')}
                          </Link>
                          <button
                            onClick={() => {
                              logout();
                              setIsUserMenuOpen(false);
                            }}
                            className="w-full flex items-center gap-2 px-3 py-2 text-sm text-slate-600 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700/50 rounded-lg transition-colors"
                          >
                            <LogOut className="w-4 h-4" />
                            {t('common.logout', 'Logout')}
                          </button>
                        </div>
                      </motion.div>
                    )}
                  </AnimatePresence>
                </div>
              )}
            </div>
          </div>

          {/* Mobile Menu Button */}
          <div className="md:hidden flex items-center gap-4">
            {/* Language Switcher */}
            <div className="relative">
              <button
                onClick={() => setIsLangMenuOpen(!isLangMenuOpen)}
                onBlur={() => setTimeout(() => setIsLangMenuOpen(false), 200)}
                className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400 transition-colors flex items-center gap-1"
              >
                <Globe className="w-5 h-5" />
                <span className="text-xs font-medium">{currentLang.toUpperCase()}</span>
              </button>
              <AnimatePresence>
                {isLangMenuOpen && (
                  <motion.div
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: 10 }}
                    transition={{ duration: 0.2 }}
                    className="absolute top-full right-0 mt-2 w-32 bg-white dark:bg-slate-800 rounded-lg shadow-2xl border border-slate-200 dark:border-slate-700 overflow-hidden z-50"
                  >
                    <button
                      onClick={() => changeLanguage('zh')}
                      className={clsx(
                        "w-full px-4 py-2 text-left text-sm transition-colors",
                        currentLang === 'zh'
                          ? "bg-sky-100 dark:bg-sky-900/30 text-sky-600 dark:text-sky-400"
                          : "hover:bg-slate-50 dark:hover:bg-slate-700/30 text-slate-600 dark:text-slate-400"
                      )}
                    >
                      中文
                    </button>
                    <button
                      onClick={() => changeLanguage('en')}
                      className={clsx(
                        "w-full px-4 py-2 text-left text-sm transition-colors",
                        currentLang === 'en'
                          ? "bg-sky-100 dark:bg-sky-900/30 text-sky-600 dark:text-sky-400"
                          : "hover:bg-slate-50 dark:hover:bg-slate-700/30 text-slate-600 dark:text-slate-400"
                      )}
                    >
                      English
                    </button>
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
            <button
              onClick={() => setIsDark(!isDark)}
              className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400 transition-colors"
            >
              {isDark ? <Sun className="w-5 h-5" /> : <Moon className="w-5 h-5" />}
            </button>
            <button className="p-2 text-slate-600 dark:text-slate-300">
              <Menu className="w-6 h-6" />
            </button>
            {/* Mobile User Menu - Only show when logged in */}
            {user && (
              <Link to="/cloud" className="w-8 h-8 rounded-full bg-gradient-to-br from-violet-500 to-purple-600 flex items-center justify-center text-white font-medium text-xs">
                {user.username.charAt(0).toUpperCase()}
              </Link>
            )}
          </div>
        </div>
      </div>
    </nav>
  );
}
