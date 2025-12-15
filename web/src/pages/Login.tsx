import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../context/AuthContext';

// GitHub Icon SVG
const GitHubIcon = () => (
  <svg className="w-5 h-5" viewBox="0 0 24 24" fill="white">
    <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
  </svg>
);

// Google Icon SVG
const GoogleIcon = () => (
  <svg className="w-5 h-5" viewBox="0 0 24 24">
    <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
    <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
    <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
    <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
  </svg>
);

// OIDC Icon SVG
const OIDCIcon = () => (
  <svg className="w-5 h-5 text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
  </svg>
);

// Cloudflare Icon SVG
const CloudflareIcon = () => (
  <svg className="w-5 h-5" viewBox="0 0 32 32" fill="none">
    <path d="M22.6 17.3l1.2-4.1c.1-.3 0-.6-.2-.8-.2-.2-.5-.3-.8-.2l-2.6.7c-.4.1-.7.4-.9.8l-.6 2.1c-.1.3.1.6.4.7l3.5-.2z" fill="#F6821F"/>
    <path d="M26.2 13.9c-.1-.4-.4-.7-.8-.7h-3.3c-.3 0-.6.2-.7.5l-.6 2c-.1.3.1.6.4.7l4 .3c.4 0 .7-.2.8-.5l.4-1.6c.1-.3 0-.6-.2-.7z" fill="#FBAD41"/>
    <path d="M8.5 17.5c-.4 0-.8-.3-.8-.7L6 12.5c-.1-.4.1-.8.5-.9l.9-.2c.2-.1.5 0 .6.2.2.2.2.4.1.6l-1 2.9h2.4l.6-2.9c.1-.4.4-.6.8-.6h1c.4 0 .7.3.7.7l-1 4.5c-.1.4-.4.6-.8.6l-2.3.1z" fill="#F6821F"/>
  </svg>
);

export default function Login() {
  const { t } = useTranslation();
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [oauthLoading, setOauthLoading] = useState<string | null>(null);
  const { login, oauthProviders, startOAuthLogin } = useAuth();
  const navigate = useNavigate();

  const hasOAuthProviders = oauthProviders.github || oauthProviders.google || 
    (oauthProviders.oidc && oauthProviders.oidc.length > 0) || oauthProviders.cloudflare;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    
    const formData = new FormData(e.target as HTMLFormElement);
    const inputPassword = (formData.get('password') as string) || password;
    
    if (!inputPassword) {
      setError(t('login.pleaseEnterPassword'));
      return;
    }
    
    setLoading(true);

    const success = await login(inputPassword);
    
    if (success) {
      // Use replace to avoid going back to login page
      navigate('/settings', { replace: true });
    } else {
      setError(t('login.invalidPassword'));
    }
    
    setLoading(false);
  };

  const handleOAuthLogin = async (provider: string) => {
    setError('');
    setOauthLoading(provider);
    
    try {
      await startOAuthLogin(provider);
    } catch (e) {
      setError(e instanceof Error ? e.message : t('login.oauthFailed'));
      setOauthLoading(null);
    }
  };

  return (
    <div className="min-h-screen relative flex items-center justify-center px-4 overflow-hidden bg-gradient-to-b from-slate-50 via-white to-slate-100 text-slate-900">
      {/* Background decoration */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute -top-24 -left-20 w-80 h-80 bg-emerald-200/50 blur-3xl rounded-full" />
        <div className="absolute -bottom-28 -right-14 w-96 h-96 bg-cyan-200/50 blur-3xl rounded-full" />
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_20%,rgba(16,185,129,0.08),transparent_40%)]" />
      </div>

      <div className="w-full max-w-md relative z-10 space-y-8">
        {/* Logo */}
        <div className="text-center space-y-3">
          <div className="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-gradient-to-br from-emerald-400/15 via-emerald-500/10 to-cyan-400/15 border border-emerald-300/60 shadow-lg shadow-emerald-500/15">
            <span className="text-4xl">⚡</span>
          </div>
          <h1 className="text-3xl font-bold text-slate-900">{t('login.title')}</h1>
          <p className="text-slate-500 text-sm">{t('login.subtitle')}</p>
        </div>

        {/* Login Card */}
        <div className="relative overflow-hidden rounded-3xl border border-white/70 bg-white/90 shadow-2xl shadow-emerald-500/10 backdrop-blur-2xl">
          <div className="absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-emerald-400 via-cyan-400 to-blue-400" />
          <div className="relative p-10 space-y-8">
            {/* OAuth Buttons */}
            {hasOAuthProviders && (
              <>
                <div className="space-y-3">
                  {oauthProviders.github && (
                    <button
                      type="button"
                      onClick={() => handleOAuthLogin('github')}
                      disabled={oauthLoading !== null}
                      className="group w-full py-3.5 px-4 rounded-xl bg-slate-900 hover:bg-slate-800 disabled:bg-slate-700 border border-slate-800 hover:border-slate-700 disabled:border-slate-700 font-medium transition-all duration-200 flex items-center justify-center gap-3 shadow-lg shadow-slate-900/30 hover:-translate-y-0.5 disabled:cursor-not-allowed"
                      style={{ color: 'white' }}
                    >
                      {oauthLoading === 'github' ? (
                        <>
                          <div 
                            className="w-5 h-5 border-2 rounded-full animate-spin" 
                            style={{ borderColor: 'rgba(255,255,255,0.2)', borderTopColor: 'white' }}
                          />
                          <span style={{ color: 'rgba(209, 213, 219, 1)' }}>{t('login.redirecting')}</span>
                        </>
                      ) : (
                        <>
                          <GitHubIcon />
                          <span style={{ color: 'white' }}>{t('login.loginWithGithub')}</span>
                          <svg className="w-4 h-4 group-hover:translate-x-0.5 transition-all ml-auto" fill="none" viewBox="0 0 24 24" stroke="rgba(156, 163, 175, 1)">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                          </svg>
                        </>
                      )}
                    </button>
                  )}

                  {oauthProviders.google && (
                    <button
                      type="button"
                      onClick={() => handleOAuthLogin('google')}
                      disabled={oauthLoading !== null}
                      className="group w-full py-3.5 px-4 rounded-xl bg-white hover:bg-slate-50 disabled:bg-slate-100 border border-slate-200 hover:border-slate-300 disabled:border-slate-200 text-slate-700 font-medium transition-all duration-200 flex items-center justify-center gap-3 shadow-lg shadow-emerald-500/10 hover:-translate-y-0.5 disabled:cursor-not-allowed"
                    >
                      {oauthLoading === 'google' ? (
                        <>
                          <div className="w-5 h-5 border-2 border-slate-200 border-t-slate-500 rounded-full animate-spin" />
                          <span className="text-slate-500">{t('login.redirecting')}</span>
                        </>
                      ) : (
                        <>
                          <GoogleIcon />
                          <span>{t('login.loginWithGoogle')}</span>
                          <svg className="w-4 h-4 text-slate-400 group-hover:text-slate-600 group-hover:translate-x-0.5 transition-all ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                          </svg>
                        </>
                      )}
                    </button>
                  )}

                  {/* OIDC Providers */}
                  {oauthProviders.oidc && oauthProviders.oidc.map((oidc) => (
                    <button
                      key={oidc.id}
                      type="button"
                      onClick={() => handleOAuthLogin(oidc.id)}
                      disabled={oauthLoading !== null}
                      className="group w-full py-3.5 px-4 rounded-xl bg-gradient-to-r from-purple-900/50 to-purple-800/50 hover:from-purple-800/50 hover:to-purple-700/50 disabled:from-purple-900/30 disabled:to-purple-800/30 border border-purple-700/50 hover:border-purple-600/50 disabled:border-purple-700/30 font-medium transition-all duration-200 flex items-center justify-center gap-3 shadow-lg shadow-purple-900/20 hover:-translate-y-0.5 disabled:cursor-not-allowed"
                      style={{ color: 'white' }}
                    >
                      {oauthLoading === oidc.id ? (
                        <>
                          <div 
                            className="w-5 h-5 border-2 rounded-full animate-spin" 
                            style={{ borderColor: 'rgba(192, 132, 252, 0.2)', borderTopColor: 'rgb(192, 132, 252)' }}
                          />
                          <span className="text-purple-300">{t('login.redirecting')}</span>
                        </>
                      ) : (
                        <>
                          <OIDCIcon />
                          <span style={{ color: 'white' }}>{oidc.name || 'SSO'}</span>
                          <svg className="w-4 h-4 text-purple-400 group-hover:text-purple-300 group-hover:translate-x-0.5 transition-all ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                          </svg>
                        </>
                      )}
                    </button>
                  ))}

                  {/* Cloudflare Access */}
                  {oauthProviders.cloudflare && (
                    <button
                      type="button"
                      onClick={() => handleOAuthLogin('cloudflare')}
                      disabled={oauthLoading !== null}
                      className="group w-full py-3.5 px-4 rounded-xl bg-gradient-to-r from-orange-500/20 to-orange-400/20 hover:from-orange-500/30 hover:to-orange-400/30 disabled:from-orange-500/10 disabled:to-orange-400/10 border border-orange-500/40 hover:border-orange-400/50 disabled:border-orange-500/20 font-medium transition-all duration-200 flex items-center justify-center gap-3 shadow-lg shadow-orange-500/10 hover:-translate-y-0.5 disabled:cursor-not-allowed"
                    >
                      {oauthLoading === 'cloudflare' ? (
                        <>
                          <div className="w-5 h-5 border-2 border-orange-300/30 border-t-orange-400 rounded-full animate-spin" />
                          <span className="text-orange-400">{t('login.redirecting')}</span>
                        </>
                      ) : (
                        <>
                          <CloudflareIcon />
                          <span className="text-orange-400">Cloudflare Access</span>
                          <svg className="w-4 h-4 text-orange-400/70 group-hover:text-orange-400 group-hover:translate-x-0.5 transition-all ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                          </svg>
                        </>
                      )}
                    </button>
                  )}
                </div>

                {/* Divider */}
                <div className="flex items-center gap-3 text-[11px] uppercase tracking-[0.2em] text-slate-400">
                  <span className="h-px flex-1 bg-gradient-to-r from-transparent via-slate-200 to-slate-200" />
                  <span className="px-3 py-1 rounded-md bg-slate-900 text-slate-100">{t('login.orUsePassword')}</span>
                  <span className="h-px flex-1 bg-gradient-to-r from-slate-200 via-slate-200 to-transparent" />
                </div>
              </>
            )}

            <form onSubmit={handleSubmit} className="space-y-6">
              <div className="space-y-2">
                <label className="block text-sm font-semibold text-slate-700">
                  {t('login.password')}
                </label>
                <input
                  type="password"
                  name="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full px-4 py-3.5 rounded-xl bg-slate-50 border border-slate-200 text-slate-900 placeholder-slate-400 focus:outline-none focus:border-emerald-500 focus:ring-4 focus:ring-emerald-500/15 transition-all"
                  placeholder={t('login.passwordPlaceholder')}
                  autoFocus={!hasOAuthProviders}
                />
              </div>

              {error && (
                <div className="p-3 rounded-xl bg-red-50 border border-red-200 text-red-600 text-sm flex items-center gap-2">
                  <svg className="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  {error}
                </div>
              )}

              <button
                type="submit"
                disabled={loading || oauthLoading !== null}
                className="w-full py-3.5 px-4 rounded-xl bg-gradient-to-r from-emerald-500 to-emerald-600 hover:from-emerald-600 hover:to-emerald-700 disabled:from-emerald-400 disabled:to-emerald-400 disabled:cursor-not-allowed text-white font-semibold transition-all duration-200 flex items-center justify-center gap-2 shadow-lg shadow-emerald-500/25 hover:shadow-emerald-500/35 hover:-translate-y-0.5"
              >
                {loading ? (
                  <>
                    <div className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                    {t('login.loggingIn')}
                  </>
                ) : (
                  <>
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1" />
                    </svg>
                    {t('login.loginButton')}
                  </>
                )}
              </button>
            </form>

            <div className="pt-4 border-t border-slate-100 text-center">
              <button
                onClick={() => navigate('/')}
                className="text-slate-500 hover:text-emerald-500 text-sm transition-colors inline-flex items-center gap-1"
              >
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
                </svg>
                {t('login.backToDashboard')}
              </button>
            </div>
          </div>
        </div>

        {/* Help text */}
        <p className="text-center text-slate-500 text-xs">
          {t('login.forgotPassword')} <code className="text-emerald-600 bg-emerald-50 px-1.5 py-0.5 rounded">sudo /opt/vstats/vstats-server --reset-password</code>
        </p>
      </div>
    </div>
  );
}
