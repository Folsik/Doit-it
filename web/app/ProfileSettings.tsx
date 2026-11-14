"use client";

import { FormEvent, useEffect, useState } from "react";
import Link from "next/link";
import { Bell, CheckCircle2, ChevronRight, Globe2, Settings2, ShieldCheck, UserRound, Wallet } from "lucide-react";

type User = { id:string; full_name:string; email:string; country_code:string; preferred_currency:string; language_code:string };
type Settings = {address_line1:string;address_line2:string;city:string;postal_code:string;residence_country:string;notifications_email:boolean;notifications_push:boolean;notifications_security:boolean;appearance:"system"|"light"|"dark"};
type Section = "personal"|"address"|"currency"|"language"|"notifications"|"appearance"|null;
const defaultSettings:Settings={address_line1:"",address_line2:"",city:"",postal_code:"",residence_country:"",notifications_email:true,notifications_push:true,notifications_security:true,appearance:"system"};
const languages=[{code:"en",name:"English"},{code:"de",name:"Deutsch"},{code:"fr",name:"Français"},{code:"es",name:"Español"}] as const;
const copy={
 en:{title:"Your profile",subtitle:"Manage your personal information and account preferences.",personal:"Personal information",address:"Country and address",currency:"Currency",language:"Language",notifications:"Notifications",appearance:"App preferences",edit:"Edit profile",name:"Legal name",country:"Country",save:"Save changes",cancel:"Cancel",private:"Changes to legal details and country require identity re-verification.",security:"Security notifications are mandatory.",notConfigured:"Identity verification is not configured yet.",details:"Personal details",preferences:"Preferences"},
 de:{title:"Dein Profil",subtitle:"Verwalte deine persönlichen Daten und Einstellungen.",personal:"Persönliche Daten",address:"Land und Adresse",currency:"Währung",language:"Sprache",notifications:"Benachrichtigungen",appearance:"App-Einstellungen",edit:"Profil bearbeiten",name:"Rechtlicher Name",country:"Land",save:"Änderungen speichern",cancel:"Abbrechen",private:"Änderungen an rechtlichen Daten und Land erfordern eine Identitätsprüfung.",security:"Sicherheitsbenachrichtigungen sind verpflichtend.",notConfigured:"Die Identitätsprüfung ist noch nicht eingerichtet.",details:"Persönliche Angaben",preferences:"Einstellungen"},
 fr:{title:"Votre profil",subtitle:"Gérez vos informations personnelles et préférences.",personal:"Informations personnelles",address:"Pays et adresse",currency:"Devise",language:"Langue",notifications:"Notifications",appearance:"Préférences",edit:"Modifier le profil",name:"Nom légal",country:"Pays",save:"Enregistrer",cancel:"Annuler",private:"Les modifications d'identité ou de pays nécessitent une nouvelle vérification.",security:"Les notifications de sécurité sont obligatoires.",notConfigured:"La vérification d'identité n'est pas encore configurée.",details:"Informations personnelles",preferences:"Préférences"},
 es:{title:"Tu perfil",subtitle:"Gestiona tus datos personales y preferencias.",personal:"Información personal",address:"País y dirección",currency:"Moneda",language:"Idioma",notifications:"Notificaciones",appearance:"Preferencias",edit:"Editar perfil",name:"Nombre legal",country:"País",save:"Guardar cambios",cancel:"Cancelar",private:"Cambiar la identidad o el país requiere verificar tu identidad.",security:"Las notificaciones de seguridad son obligatorias.",notConfigured:"La verificación de identidad aún no está configurada.",details:"Datos personales",preferences:"Preferencias"}
} as const;
type Locale=keyof typeof copy;
async function api<T>(path:string,method:"GET"|"PUT"|"PATCH"|"POST"="GET",body?:unknown):Promise<T>{
 const r=await fetch("/api/banking"+path,{method,credentials:"same-origin",headers:{"Content-Type":"application/json"},body:body===undefined?undefined:JSON.stringify(body),cache:"no-store"});
 const result=await r.json().catch(()=>({}));
 if(!r.ok) throw Object.assign(new Error(result?.error?.message??(r.status===401?"Your session has expired. Please sign in again.":r.status===404?"Profile API route was not found. Verify frontend and API versions.":`Profile request failed (HTTP ${r.status}).`)),{status:r.status});
 return result as T;
}
export default function ProfileSettings(){
 const [user,setUser]=useState<User|null>(null),[settings,setSettings]=useState<Settings>(defaultSettings);
 const [section,setSection]=useState<Section>(null),[loading,setLoading]=useState(true),[busy,setBusy]=useState(false),[unauthorized,setUnauthorized]=useState(false);
 const [error,setError]=useState(""),[notice,setNotice]=useState("");
 const locale:Locale=(user?.language_code && user.language_code in copy ? user.language_code : "en") as Locale;
 const t=copy[locale];
 useEffect(()=>{
   const media=window.matchMedia("(prefers-color-scheme: dark)");
   const apply=()=>{document.documentElement.dataset.doitTheme=settings.appearance==="system"?(media.matches?"dark":"light"):settings.appearance};
   apply();
   media.addEventListener("change",apply);
   return ()=>media.removeEventListener("change",apply);
 },[settings.appearance]);
 useEffect(()=>{
   let alive=true;
   Promise.all([api<User>("/v1/me"),api<Settings>("/v1/me/settings")]).then(([u,s])=>{if(alive){setUser(u);setSettings(s);document.documentElement.lang=u.language_code in copy?u.language_code:"en";}}).catch((e:Error&{status?:number})=>{if(alive){setUnauthorized(e.status===401);setError(e.message);}}).finally(()=>{if(alive)setLoading(false)});
   return ()=>{alive=false};
 },[]);
 async function save(method:"PUT"|"PATCH",path:string,body:unknown){
  setBusy(true);setError("");setNotice("");
  try{
   if(path==="/v1/me"){const u=await api<User>(path,method,body);setUser(u);document.documentElement.lang=u.language_code in copy?u.language_code:"en";}
   else {const s=await api<Settings>(path,method,body);setSettings(s);}
   setNotice("Changes saved");setSection(null);
  }catch(e){setError(e instanceof Error?e.message:"Could not save settings");}
  finally{setBusy(false);}
 }
 async function requestVerification(){
  setBusy(true);setError("");setNotice("");
  try{await api("/v1/me/identity/reverification","POST",{});}
  catch(e){setNotice(e instanceof Error?e.message:t.notConfigured);}
  finally{setBusy(false);}
 }
 if(loading)return <section className="panel profile-settings-status" role="status">Loading profile…</section>;
 if(unauthorized)return <section className="panel profile-settings-status"><h2>Sign in to manage your profile</h2><p>Your account settings are only available after authentication.</p><Link className="primary-button" href="/auth">Sign in or create an account</Link></section>;
 if(!user)return <section className="panel profile-settings-status" role="alert">{error||"Unable to load your profile"} <button className="secondary-button compact" onClick={()=>window.location.reload()}>Retry</button></section>;
 const initials=user.full_name?user.full_name.trim().split(/\s+/).slice(0,2).map(v=>v[0]).join("").toUpperCase():"U";
 const items:{id:Exclude<Section,null>;icon:typeof UserRound;title:string;desc:string}[]=[
 {id:"personal",icon:UserRound,title:t.personal,desc:user.full_name||"Identity information"},
 {id:"address",icon:Globe2,title:t.address,desc:[settings.residence_country||user.country_code,settings.city].filter(Boolean).join(" · ")||"Add your address"},
 {id:"currency",icon:Wallet,title:t.currency,desc:"Euro (EUR)"},
 {id:"language",icon:Settings2,title:t.language,desc:languages.find(x=>x.code===user.language_code)?.name||"English"},
 {id:"notifications",icon:Bell,title:t.notifications,desc:"Email, push and security alerts"},
 {id:"appearance",icon:Settings2,title:t.appearance,desc:settings.appearance}
 ];
 const editor=section;
 const row=(id:Exclude<Section,null>)=>{const item=items.find(x=>x.id===id)!;const Icon=item.icon;return <button type="button" className="setting-row setting-button" key={id} onClick={()=>{setSection(id);setError("");setNotice("");}}><span className="setting-icon"><Icon size={18}/></span><span className="setting-copy"><strong>{item.title}</strong><small>{item.desc}</small></span><ChevronRight size={17} className="setting-chevron"/></button>};
 return <>
  <div className="welcome-row"><div><p className="eyebrow">YOUR SPACE</p><h1>{t.title}</h1><p className="muted">{t.subtitle}</p></div></div>
  <article className="panel profile-card"><div className="profile-large-avatar">{initials}</div><div className="profile-name"><h2>{user.full_name||"Account holder"}</h2><p>{user.email||"No email added"}</p></div><button className="secondary-button compact" onClick={()=>setSection("personal")}>{t.edit}</button></article>
  <article className="panel settings-panel"><h2 className="profile-section-title">{t.details}</h2>{items.slice(0,4).map(i=>row(i.id))}</article>
  <article className="panel settings-panel"><h2 className="profile-section-title">{t.preferences}</h2>{items.slice(4).map(i=>row(i.id))}</article>
  {notice&&<p role="status" className="profile-notice">{notice}</p>}
  {editor&&<div className="modal-backdrop" onMouseDown={e=>{if(e.target===e.currentTarget&&!busy)setSection(null)}}><section className="transfer-modal profile-editor" role="dialog" aria-modal="true" aria-labelledby="profile-editor-title">
   <div className="profile-editor-heading"><h2 id="profile-editor-title">{items.find(x=>x.id===editor)?.title}</h2><button type="button" className="secondary-button compact" onClick={()=>setSection(null)} disabled={busy}>{t.cancel}</button></div>
   {error&&<p className="auth-alert error" role="alert">{error}</p>}
   {editor==="personal"&&<div className="profile-editor-form"><label>{t.name}<input value={user.full_name} readOnly disabled/></label><label>Email<input value={user.email||""} readOnly disabled/></label><label>{t.country}<input value={user.country_code||"Not set"} readOnly disabled/></label><p className="auth-hint">{t.private}</p><button className="primary-button" type="button" disabled={busy} onClick={requestVerification}><ShieldCheck size={17}/> Request identity re-verification</button></div>}
   {editor==="address"&&<form className="profile-editor-form" onSubmit={(e:FormEvent)=>{e.preventDefault();void save("PUT","/v1/me/address",{address_line1:settings.address_line1,address_line2:settings.address_line2,city:settings.city,postal_code:settings.postal_code});}}>
    <label>Country (requires identity verification)<input value={settings.residence_country||user.country_code||"Not set"} readOnly disabled/></label>
    <button type="button" className="secondary-button compact" onClick={requestVerification} disabled={busy}>Request country change via Veriff</button>
    <label>Street address<input maxLength={200} value={settings.address_line1} onChange={e=>setSettings(s=>({...s,address_line1:e.target.value}))}/></label>
    <label>Apartment / additional address<input maxLength={200} value={settings.address_line2} onChange={e=>setSettings(s=>({...s,address_line2:e.target.value}))}/></label>
    <label>City<input maxLength={120} value={settings.city} onChange={e=>setSettings(s=>({...s,city:e.target.value}))}/></label>
    <label>Postal code<input maxLength={32} value={settings.postal_code} onChange={e=>setSettings(s=>({...s,postal_code:e.target.value}))}/></label>
    <button className="primary-button" disabled={busy}>{busy?"Saving…":t.save}</button>
   </form>}
   {editor==="currency"&&<div className="profile-editor-form"><label>Account currency<select value="EUR" disabled><option value="EUR">Euro (EUR)</option></select></label><p className="auth-hint">EUR is the only supported currency. Currency exchange and multi-currency accounts are not available yet; changing a display preference would not convert your balance.</p><button className="secondary-button" onClick={()=>setSection(null)}>Close</button></div>}
   {editor==="language"&&<form className="profile-editor-form" onSubmit={e=>{e.preventDefault();void save("PATCH","/v1/me",{language_code:user.language_code});}}>
     <label>{t.language}<select value={user.language_code} onChange={e=>setUser(u=>u?{...u,language_code:e.target.value}:u)}>{languages.map(l=><option key={l.code} value={l.code}>{l.name}</option>)}</select></label>
     <p className="auth-hint">Profile labels are translated. The rest of the dashboard is being localized separately.</p>
     <button className="primary-button" disabled={busy}>{busy?"Saving…":t.save}</button>
   </form>}
   {editor==="notifications"&&<form className="profile-editor-form" onSubmit={e=>{e.preventDefault();void save("PUT","/v1/me/notifications",{email:settings.notifications_email,push:settings.notifications_push,security:true});}}>
     <label className="profile-toggle"><input type="checkbox" checked={settings.notifications_email} onChange={e=>setSettings(s=>({...s,notifications_email:e.target.checked}))}/> Email notifications</label>
     <label className="profile-toggle"><input type="checkbox" checked={settings.notifications_push} onChange={e=>setSettings(s=>({...s,notifications_push:e.target.checked}))}/> Push notifications</label>
     <label className="profile-toggle"><input type="checkbox" checked readOnly disabled/> Security notifications (required)</label>
     <p className="auth-hint">{t.security} Delivery channels are not yet integrated; these preferences control future delivery.</p>
     <button className="primary-button" disabled={busy}>{busy?"Saving…":t.save}</button>
   </form>}
   {editor==="appearance"&&<form className="profile-editor-form" onSubmit={e=>{e.preventDefault();void save("PUT","/v1/me/appearance",{appearance:settings.appearance});}}>
     <label>Appearance<select value={settings.appearance} onChange={e=>setSettings(s=>({...s,appearance:e.target.value as Settings["appearance"]}))}><option value="system">System</option><option value="light">Light</option><option value="dark">Dark</option></select></label>
     <p className="auth-hint">The selected preference is saved to your account. Full dark-mode styling for every dashboard page is still being developed.</p>
     <button className="primary-button" disabled={busy}>{busy?"Saving…":t.save}</button>
   </form>}
  </section></div>}
 </>;
}
