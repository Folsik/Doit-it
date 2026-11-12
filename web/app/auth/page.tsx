"use client";

import { FormEvent, useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeft, ArrowRight, CheckCircle2, ShieldCheck } from "lucide-react";

type Country = { code: string; name: string; calling_code: string };
type Step = "welcome" | "phone" | "otp" | "pin" | "email" | "card" | "login" | "done";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

async function request(path: string, body?: unknown, token?: string) {
  const response = await fetch(API + path, {
    method: body === undefined ? "GET" : "POST",
    headers: { "Content-Type": "application/json", ...(token ? { Authorization: "Bearer " + token } : {}) },
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    cache: "no-store",
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(data?.error?.message ?? "Something went wrong. Please try again.");
  return data;
}

export default function AuthPage() {
  const [step, setStep] = useState<Step>("welcome");
  const [countries, setCountries] = useState<Country[]>([]);
  const [country, setCountry] = useState("DE");
  const [phone, setPhone] = useState("");
  const [fullPhone, setFullPhone] = useState("");
  const [challenge, setChallenge] = useState("");
  const [otp, setOtp] = useState("");
  const [devCode, setDevCode] = useState("");
  const [verificationToken, setVerificationToken] = useState("");
  const [pin, setPin] = useState("");
  const [confirmPin, setConfirmPin] = useState("");
  const [email, setEmail] = useState("");
  const [accessToken, setAccessToken] = useState("");
  const [scheme, setScheme] = useState("visa");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  useEffect(() => {
    fetch(API + "/v1/onboarding/countries", { cache: "no-store" })
      .then(r => r.json()).then(data => setCountries(data.items ?? []))
      .catch(() => setError("Could not load supported countries. Check that the Go API is running."));
  }, []);

  const selectedCountry = countries.find(c => c.code === country);
  const phoneE164 = selectedCountry ? selectedCountry.calling_code + phone.replace(/\D/g, "").replace(/^0+/, "") : "";

  async function run(action: () => Promise<void>) {
    setBusy(true); setError(""); setMessage("");
    try { await action(); } catch (e) { setError(e instanceof Error ? e.message : "Request failed"); }
    finally { setBusy(false); }
  }

  async function sendCode(e: FormEvent) {
    e.preventDefault();
    await run(async () => {
      const data = await request("/v1/onboarding/send-otp", { country_code: country, phone_e164: phoneE164 });
      setFullPhone(phoneE164); setChallenge(data.challenge_id); setDevCode(data.development_code ?? "");
      setStep("otp"); setMessage(data.development_code ? "Development verification code generated. It expires in five minutes." : "Verification initiated. Check your phone for the code.");
    });
  }

  async function verifyCode(e: FormEvent) {
    e.preventDefault();
    await run(async () => {
      const data = await request("/v1/onboarding/verify-otp", { challenge_id: challenge, code: otp });
      setVerificationToken(data.verification_token); setStep("pin");
    });
  }

  async function createAccount(e: FormEvent) {
    e.preventDefault();
    if (pin !== confirmPin) { setError("The PINs do not match."); return; }
    await run(async () => {
      const data = await request("/v1/onboarding/set-pin", { verification_token: verificationToken, pin });
      setAccessToken(data.access_token); setPin(""); setConfirmPin(""); setStep("email");
      setMessage("Your account has been created. You can add an email or skip this step.");
    });
  }

  async function saveEmail(e: FormEvent) {
    e.preventDefault();
    await run(async () => {
      if (email.trim()) await fetch(API + "/v1/me/email", {
        method: "PUT", headers: { "Content-Type": "application/json", Authorization: "Bearer " + accessToken },
        body: JSON.stringify({ email: email.trim() }),
      }).then(async r => { if (!r.ok) { const d = await r.json(); throw new Error(d?.error?.message ?? "Could not save email"); } });
      setStep("card");
    });
  }

  async function chooseCard(e: FormEvent) {
    e.preventDefault();
    await run(async () => {
      await request("/v1/cards/choose", { scheme }, accessToken);
      setStep("done");
    });
  }

  async function login(e: FormEvent) {
    e.preventDefault();
    await run(async () => {
      const data = await request("/v1/auth/phone-login", { phone_e164: phoneE164 || fullPhone, pin });
      setAccessToken(data.access_token); setPin(""); setStep("done");
    });
  }

  const title: Record<Step, string> = {
    welcome: "Your money, moving with you", phone: "Create your account", otp: "Verify your phone",
    pin: "Create your app PIN", email: "One last detail", card: "Choose your card",
    login: "Welcome back", done: "You're all set",
  };

  return <main className="auth-page">
    <div className="auth-top"><Link href="/" className="auth-brand"><span className="auth-mark">d</span> doit</Link><span className="auth-secure"><ShieldCheck size={15}/> Secure onboarding</span></div>
    <section className="auth-card">
      <div className="auth-progress"><span style={{ width: ({welcome:"0%",phone:"20%",otp:"40%",pin:"60%",email:"75%",card:"90%",login:"100%",done:"100%"} as Record<Step,string>)[step] }}/></div>
      <div className="auth-kicker">DOIT · PERSONAL FINANCE</div>
      <h1>{title[step]}</h1>
      <p className="auth-subtitle">{step === "welcome" ? "A simpler way to look after your everyday finances." : step === "done" ? "Your prototype account is ready. No real card or payment network has been activated." : "A few quick steps and you're ready to go."}</p>
      {error && <div className="auth-alert error" role="alert">{error}</div>}
      {message && <div className="auth-alert info">{message}</div>}
      {step === "welcome" && <div className="auth-actions">
        <button className="auth-primary" onClick={() => { setError(""); setStep("phone"); }}>Create a new account <ArrowRight size={17}/></button>
        <button className="auth-secondary" onClick={() => { setError(""); setStep("login"); }}>I already have an account</button>
        <p className="auth-footnote">Development onboarding uses a test OTP. Real SMS delivery is not configured.</p>
      </div>}
      {step === "phone" && <form onSubmit={sendCode} className="auth-form">
        <label>Country or region<select value={country} onChange={e=>setCountry(e.target.value)} required>{countries.map(c=><option key={c.code} value={c.code}>{c.name} ({c.calling_code})</option>)}</select></label>
        <label>Phone number<input value={phone} onChange={e=>setPhone(e.target.value)} inputMode="tel" placeholder="1512345678" required minLength={6}/></label>
        <p className="auth-hint">Only regions supported by the backend are listed. Final eligibility still requires identity and compliance checks.</p>
        <button className="auth-primary" disabled={busy || !selectedCountry}>{busy ? "Sending…" : "Send verification code"} <ArrowRight size={17}/></button>
        <button type="button" className="auth-back" onClick={()=>setStep("welcome")}><ArrowLeft size={15}/> Back</button>
      </form>}
      {step === "otp" && <form onSubmit={verifyCode} className="auth-form">
        <p className="auth-hint">Enter the six-digit code for <strong>{fullPhone}</strong>.</p>
        {devCode && <div className="dev-code"><span>Local development code</span><strong>{devCode}</strong></div>}
        <label>Verification code<input value={otp} onChange={e=>setOtp(e.target.value.replace(/\D/g,"").slice(0,6))} inputMode="numeric" autoComplete="one-time-code" placeholder="000000" required minLength={6} maxLength={6}/></label>
        <button className="auth-primary" disabled={busy || otp.length!==6}>{busy ? "Verifying…" : "Verify phone"} <ArrowRight size={17}/></button>
        <button type="button" className="auth-back" onClick={()=>setStep("phone")}><ArrowLeft size={15}/> Change phone number</button>
      </form>}
      {step === "pin" && <form onSubmit={createAccount} className="auth-form">
        <label>Create six-digit app PIN<input type="password" value={pin} onChange={e=>setPin(e.target.value.replace(/\D/g,"").slice(0,6))} inputMode="numeric" autoComplete="new-password" placeholder="••••••" required minLength={6} maxLength={6}/></label>
        <label>Confirm app PIN<input type="password" value={confirmPin} onChange={e=>setConfirmPin(e.target.value.replace(/\D/g,"").slice(0,6))} inputMode="numeric" autoComplete="new-password" placeholder="••••••" required minLength={6} maxLength={6}/></label>
        <p className="auth-hint">Avoid repeated or sequential digits. This is your sign-in PIN, not a separate authenticator-app 2FA code.</p>
        <button className="auth-primary" disabled={busy || pin.length!==6 || confirmPin.length!==6}>{busy ? "Creating account…" : "Create account"} <ArrowRight size={17}/></button>
      </form>}
      {step === "email" && <form onSubmit={saveEmail} className="auth-form">
        <label>Email address <span className="auth-optional">Optional</span><input type="email" value={email} onChange={e=>setEmail(e.target.value)} placeholder="you@example.com"/></label>
        <p className="auth-hint">You can add an email now or skip and add it later.</p>
        <button className="auth-primary" disabled={busy}>{busy ? "Saving…" : "Continue"} <ArrowRight size={17}/></button>
        <button type="button" className="auth-back" onClick={()=>setStep("card")}>Skip for now</button>
      </form>}
      {step === "card" && <form onSubmit={chooseCard} className="auth-form">
        <div className="scheme-options">
          <button type="button" className={scheme==="visa"?"scheme-choice chosen":"scheme-choice"} onClick={()=>setScheme("visa")}><strong>VISA</strong><span>Debit card</span></button>
          <button type="button" className={scheme==="mastercard"?"scheme-choice chosen":"scheme-choice"} onClick={()=>setScheme("mastercard")}><strong>Mastercard</strong><span>Debit card</span></button>
        </div>
        <p className="auth-hint">This creates a prototype account and records your card preference. It does not issue a real payment card.</p>
        <button className="auth-primary" disabled={busy}>{busy ? "Opening account…" : "Finish setup"} <ArrowRight size={17}/></button>
      </form>}
      {step === "login" && <form onSubmit={login} className="auth-form">
        <label>Country<select value={country} onChange={e=>setCountry(e.target.value)} required>{countries.map(c=><option key={c.code} value={c.code}>{c.name} ({c.calling_code})</option>)}</select></label>
        <label>Phone number<input value={phone} onChange={e=>setPhone(e.target.value)} inputMode="tel" placeholder="1512345678" required/></label>
        <label>Six-digit app PIN<input type="password" value={pin} onChange={e=>setPin(e.target.value.replace(/\D/g,"").slice(0,6))} inputMode="numeric" autoComplete="current-password" placeholder="••••••" required minLength={6} maxLength={6}/></label>
        <button className="auth-primary" disabled={busy || pin.length!==6}>{busy ? "Signing in…" : "Sign in"} <ArrowRight size={17}/></button>
        <button type="button" className="auth-back" onClick={()=>setStep("welcome")}><ArrowLeft size={15}/> Back</button>
      </form>}
      {step === "done" && <div className="auth-actions">
        <div className="auth-success"><CheckCircle2 size={22}/><span>Account setup complete</span></div>
        <p className="auth-hint">The Go API accepted your request. The dashboard remains a separate demonstration, not an authenticated banking session.</p>
        <Link href="/" className="auth-primary">View demo dashboard <ArrowRight size={17}/></Link>
      </div>}
      <div className="auth-bottom"><span>Protected with secure verification</span><span>Prototype · No real funds</span></div>
    </section>
    <p className="auth-legal">By continuing, you agree to the prototype terms. This is not a live banking service.</p>
  </main>;
}
