"use client";

import { FormEvent, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowLeft, ArrowRight, CheckCircle2, ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";

type Country = { code: string; name: string; calling_code: string };
type Step = "welcome" | "phone" | "otp" | "pin" | "email" | "card" | "login" | "done";

const API = "/api";

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
  const router = useRouter();
  const [step, setStep] = useState<Step>("welcome");
  const [countries, setCountries] = useState<Country[]>([]);
  const [countriesLoading, setCountriesLoading] = useState(true);
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
      .then(async r => { if (!r.ok) throw new Error(`Countries API returned ${r.status}`); return r.json(); })
      .then(data => { if (!Array.isArray(data.items) || data.items.length === 0) throw new Error("Empty country list"); setCountries(data.items); setCountry(previous => data.items.some((item: Country) => item.code === previous) ? previous : data.items[0].code); setError(""); })
      .catch(() => setError("Could not load supported countries. Please check the API connection and try again."))
      .finally(() => setCountriesLoading(false));
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
      await activateDashboard(accessToken);
    });
  }

  async function activateDashboard(token: string) {
    const response = await fetch("/api/session", { method: "POST", headers: {"Content-Type":"application/json"}, body: JSON.stringify({ access_token: token }), cache: "no-store" });
    if (!response.ok) {
      const data=await response.json().catch(()=>({}));
      throw new Error(data?.error ?? "Could not activate your account session");
    }
    router.push("/?view=Profile");
  }

  async function login(e: FormEvent) {
    e.preventDefault();
    await run(async () => {
      const data = await request("/v1/auth/phone-login", { phone_e164: phoneE164 || fullPhone, pin });
      setAccessToken(data.access_token); setPin(""); await activateDashboard(data.access_token);
    });
  }

  const title: Record<Step, string> = {
    welcome: "Your money, moving with you", phone: "Create your account", otp: "Verify your phone",
    pin: "Create your app PIN", email: "One last detail", card: "Choose your card",
    login: "Welcome back", done: "You're all set",
  };

  const form = "flex flex-col gap-5";
  const field = "flex flex-col gap-2";
  const primary = "w-full";
  const heading = title[step];
  const stepWidth:Record<Step,string>={welcome:"5%",phone:"20%",otp:"40%",pin:"60%",email:"75%",card:"90%",login:"90%",done:"100%"};
  return <main className="min-h-screen bg-background px-4 py-7 sm:px-6 sm:py-12">
    <div className="mx-auto mb-10 flex max-w-5xl items-center justify-between">
      <Link href="/" className="flex items-center gap-2 text-2xl font-extrabold tracking-tight text-foreground"><span className="grid size-9 place-items-center rounded-xl bg-primary text-xl text-primary-foreground">d</span> doit</Link>
      <span className="inline-flex items-center gap-2 text-xs font-semibold text-muted-foreground"><ShieldCheck size={16}/> Secure onboarding</span>
    </div>
    <Card className="mx-auto w-full max-w-[510px] p-6 shadow-[0_20px_70px_rgba(27,53,91,.07)] sm:p-10">
      <div className="mb-8 h-1.5 overflow-hidden rounded-full bg-muted"><div className="h-full rounded-full bg-primary transition-all duration-300" style={{width:stepWidth[step]}}/></div>
      <p className="mb-3 text-[11px] font-bold uppercase tracking-[.2em] text-muted-foreground">Doit · Personal finance</p>
      <h1 className="font-display text-3xl font-extrabold tracking-tight text-foreground sm:text-4xl">{heading}</h1>
      <p className="mb-7 mt-3 text-sm leading-6 text-muted-foreground">{step==="welcome"?"A simpler way to look after your everyday finances.":step==="done"?"Your prototype account is ready. No real payment network has been activated.":"A few quick steps and you’re ready to go."}</p>
      {error&&<div role="alert" className="mb-5 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">{error}</div>}
      {message&&<div role="status" className="mb-5 rounded-xl border border-blue-200 bg-blue-50 p-4 text-sm text-blue-800">{message}</div>}
      {step==="welcome"&&<div className="flex flex-col gap-3">
        <Button size="lg" className={primary} onClick={()=>{setError("");setStep("phone")}}>Create a new account <ArrowRight size={17}/></Button>
        <Button size="lg" variant="outline" className={primary} onClick={()=>{setError("");setStep("login")}}>I already have an account</Button>
        <p className="mt-2 text-xs leading-5 text-muted-foreground">Development onboarding uses a test OTP. Real SMS delivery is not configured.</p>
      </div>}
      {step==="phone"&&<form onSubmit={sendCode} className={form}>
        <div className={field}><Label htmlFor="region">Country or region</Label><Select id="region" value={countries.length?country:""} disabled={countriesLoading||!countries.length} onChange={e=>setCountry(e.target.value)} required><option value="" disabled>{countriesLoading?"Loading countries…":"Select a country"}</option>{countries.map(c=><option key={c.code} value={c.code}>{c.name} ({c.calling_code})</option>)}</Select></div>
        <div className={field}><Label htmlFor="phone">Phone number</Label><Input id="phone" value={phone} onChange={e=>setPhone(e.target.value)} inputMode="tel" placeholder="1512345678" required minLength={6}/></div>
        <p className="text-xs leading-5 text-muted-foreground">Availability is based on supported regions. KYC and compliance checks are separate.</p>
        <Button size="lg" disabled={busy||!selectedCountry}>{busy?"Sending…":"Send verification code"} <ArrowRight size={17}/></Button>
        <Button type="button" variant="ghost" onClick={()=>setStep("welcome")}><ArrowLeft size={16}/> Back</Button>
      </form>}
      {step==="otp"&&<form onSubmit={verifyCode} className={form}>
        <p className="text-sm text-muted-foreground">Enter the code for <strong className="text-foreground">{fullPhone}</strong>.</p>
        {devCode&&<div className="flex items-center justify-between rounded-xl border border-dashed border-blue-300 bg-blue-50 p-4 text-blue-700"><span className="text-xs">Development OTP</span><strong className="font-mono text-xl tracking-[.2em]">{devCode}</strong></div>}
        <div className={field}><Label htmlFor="otp">Verification code</Label><Input id="otp" value={otp} onChange={e=>setOtp(e.target.value.replace(/\D/g,"").slice(0,6))} inputMode="numeric" autoComplete="one-time-code" placeholder="000000" required minLength={6} maxLength={6}/></div>
        <Button size="lg" disabled={busy||otp.length!==6}>{busy?"Verifying…":"Verify phone"} <ArrowRight size={17}/></Button>
        <Button type="button" variant="ghost" onClick={()=>setStep("phone")}><ArrowLeft size={16}/> Change phone number</Button>
      </form>}
      {step==="pin"&&<form onSubmit={createAccount} className={form}>
        <div className={field}><Label htmlFor="pin">Create six-digit PIN</Label><Input id="pin" type="password" value={pin} onChange={e=>setPin(e.target.value.replace(/\D/g,"").slice(0,6))} inputMode="numeric" autoComplete="new-password" required minLength={6} maxLength={6}/></div>
        <div className={field}><Label htmlFor="confirm-pin">Confirm PIN</Label><Input id="confirm-pin" type="password" value={confirmPin} onChange={e=>setConfirmPin(e.target.value.replace(/\D/g,"").slice(0,6))} inputMode="numeric" autoComplete="new-password" required minLength={6} maxLength={6}/></div>
        <p className="text-xs text-muted-foreground">Avoid repeated or sequential digits.</p>
        <Button size="lg" disabled={busy||pin.length!==6||confirmPin.length!==6}>{busy?"Creating…":"Create account"} <ArrowRight size={17}/></Button>
      </form>}
      {step==="email"&&<form onSubmit={saveEmail} className={form}>
        <div className={field}><Label htmlFor="email">Email address (optional)</Label><Input id="email" type="email" value={email} onChange={e=>setEmail(e.target.value)} placeholder="you@example.com"/></div>
        <Button size="lg" disabled={busy}>{busy?"Saving…":"Continue"} <ArrowRight size={17}/></Button>
        <Button type="button" variant="ghost" onClick={()=>setStep("card")}>Skip for now</Button>
      </form>}
      {step==="card"&&<form onSubmit={chooseCard} className={form}>
        <div className="grid grid-cols-2 gap-3">
          {(["visa","mastercard"] as const).map(s=><button key={s} type="button" onClick={()=>setScheme(s)} aria-pressed={scheme===s} className={"flex min-h-28 flex-col justify-between rounded-xl border p-4 text-left transition-colors "+(scheme===s?"border-primary bg-secondary ring-2 ring-primary/15":"border-border bg-card hover:bg-accent")}><strong className="text-lg uppercase">{s==="visa"?"VISA":"Mastercard"}</strong><span className="text-xs text-muted-foreground">Debit card · prototype</span></button>)}
        </div>
        <p className="text-xs leading-5 text-muted-foreground">Only a prototype card preference is recorded. No real card is issued.</p>
        <Button size="lg" disabled={busy}>{busy?"Opening account…":"Finish setup"} <ArrowRight size={17}/></Button>
      </form>}
      {step==="login"&&<form onSubmit={login} className={form}>
        <div className={field}><Label htmlFor="login-region">Country or region</Label><Select id="login-region" value={countries.length?country:""} onChange={e=>setCountry(e.target.value)} required disabled={countriesLoading||!countries.length}><option value="" disabled>{countriesLoading?"Loading countries…":"Select a country"}</option>{countries.map(c=><option key={c.code} value={c.code}>{c.name} ({c.calling_code})</option>)}</Select></div>
        <div className={field}><Label htmlFor="login-phone">Phone number</Label><Input id="login-phone" value={phone} onChange={e=>setPhone(e.target.value)} inputMode="tel" placeholder="1512345678" required/></div>
        <div className={field}><Label htmlFor="login-pin">Six-digit app PIN</Label><Input id="login-pin" type="password" value={pin} onChange={e=>setPin(e.target.value.replace(/\D/g,"").slice(0,6))} inputMode="numeric" autoComplete="current-password" required minLength={6} maxLength={6}/></div>
        <Button size="lg" disabled={busy||pin.length!==6||!selectedCountry}>{busy?"Signing in…":"Sign in"} <ArrowRight size={17}/></Button>
        <Button type="button" variant="ghost" onClick={()=>setStep("welcome")}><ArrowLeft size={16}/> Back</Button>
      </form>}
      {step==="done"&&<div className="space-y-5 text-center"><CheckCircle2 size={36} className="mx-auto text-emerald-500"/><p className="text-sm text-muted-foreground">Account created successfully. Sign in to manage your account.</p><Button asChild size="lg" className="w-full"><Link href="/?view=Profile">Open your profile <ArrowRight size={16}/></Link></Button></div>}
      <div className="mt-8 flex items-center justify-between border-t border-border pt-5 text-[11px] text-muted-foreground"><span>Protected with secure verification</span><span>Prototype · No real funds</span></div>
    </Card>
    <p className="mx-auto mt-6 max-w-md text-center text-xs leading-5 text-muted-foreground">By continuing, you agree to the prototype terms. This is not a live banking service.</p>
  </main>;
}
