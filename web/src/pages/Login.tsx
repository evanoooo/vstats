import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

// GitHub Icon SVG
const GitHubIcon = () => (
  <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
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

export default function Login() {
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [oauthLoading, setOauthLoading] = useState<string | null>(null);
  const { login, oauthProviders, startOAuthLogin } = useAuth();
  const navigate = useNavigate();

  const hasOAuthProviders = oauthProviders.github || oauthProviders.google;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    
    const formData = new FormData(e.target as HTMLFormElement);
    const inputPassword = (formData.get('password') as string) || password;
    
    if (!inputPassword) {
      setError('Please enter a password');
      return;
    }
    
    setLoading(true);

    const success = await login(inputPassword);
    
    if (success) {
      navigate('/settings');
    } else {
      setError('Invalid password');
    }
    
    setLoading(false);
  };

  const handleOAuthLogin = async (provider: 'github' | 'google') => {
    setError('');
    setOauthLoading(provider);
    
    try {
      await startOAuthLogin(provider);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'OAuth login failed');
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
          <h1 className="text-3xl font-bold text-slate-900">vStats</h1>
          <p className="text-slate-500 text-sm">服务器监控管理面板</p>
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
                      className="group w-full py-3.5 px-4 rounded-xl bg-slate-900 hover:bg-slate-800 disabled:bg-slate-700 border border-slate-800 hover:border-slate-700 disabled:border-slate-700 text-white font-medium transition-all duration-200 flex items-center justify-center gap-3 shadow-lg shadow-slate-900/30 hover:-translate-y-0.5 disabled:cursor-not-allowed"
                    >
                      {oauthLoading === 'github' ? (
                        <>
                          <div className="w-5 h-5 border-2 border-white/20 border-t-white rounded-full animate-spin" />
                          <span className="text-gray-300">正在跳转...</span>
                        </>
                      ) : (
                        <>
                          <GitHubIcon />
                          <span>使用 GitHub 登录</span>
                          <svg className="w-4 h-4 text-gray-400 group-hover:text-white group-hover:translate-x-0.5 transition-all ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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
                          <span className="text-slate-500">正在跳转...</span>
                        </>
                      ) : (
                        <>
                          <GoogleIcon />
                          <span>使用 Google 登录</span>
                          <svg className="w-4 h-4 text-slate-400 group-hover:text-slate-600 group-hover:translate-x-0.5 transition-all ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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
                  <span className="px-3 py-1 rounded-md bg-slate-900 text-slate-100">或使用密码</span>
                  <span className="h-px flex-1 bg-gradient-to-r from-slate-200 via-slate-200 to-transparent" />
                </div>
              </>
            )}

            <form onSubmit={handleSubmit} className="space-y-6">
              <div className="space-y-2">
                <label className="block text-sm font-semibold text-slate-700">
                  管理员密码
                </label>
                <input
                  type="password"
                  name="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full px-4 py-3.5 rounded-xl bg-slate-50 border border-slate-200 text-slate-900 placeholder-slate-400 focus:outline-none focus:border-emerald-500 focus:ring-4 focus:ring-emerald-500/15 transition-all"
                  placeholder="输入密码"
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
                    登录中...
                  </>
                ) : (
                  <>
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1" />
                    </svg>
                    登录
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
                返回仪表盘
              </button>
            </div>
          </div>
        </div>

        {/* Help text */}
        <p className="text-center text-slate-500 text-xs">
          忘记密码？运行 <code className="text-emerald-600 bg-emerald-50 px-1.5 py-0.5 rounded">./vstats-server --reset-password</code>
        </p>
      </div>
    </div>
  );
}
