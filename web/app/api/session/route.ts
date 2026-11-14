import { NextRequest, NextResponse } from "next/server";

export const runtime = "nodejs";
const cookieName = "doit_session";
function validOrigin(request: NextRequest) {
  const origin = request.headers.get("origin");
  return !!origin && origin === request.nextUrl.origin;
}
export async function POST(request: NextRequest) {
  if (!validOrigin(request)) return NextResponse.json({error:"Invalid origin"}, {status:403});
  const contentType=request.headers.get("content-type") ?? "";
  if (!contentType.includes("application/json")) return NextResponse.json({error:"JSON required"}, {status:415});
  const data = await request.json().catch(()=>null);
  const token = data?.access_token;
  if (typeof token !== "string" || token.length < 50 || token.length > 4096) return NextResponse.json({error:"Invalid token"}, {status:400});
  // Validate token against the Go API before installing the session.
  const base = process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080";
  try {
    const check = await fetch(base + "/v1/me", {headers:{Authorization:"Bearer "+token},cache:"no-store",signal:AbortSignal.timeout(8000)});
    if (!check.ok) return NextResponse.json({error:"Sign-in is invalid or expired"}, {status:401});
  } catch {return NextResponse.json({error:"Authentication server unavailable"}, {status:503});}
  const response=NextResponse.json({ok:true});
  response.cookies.set(cookieName,token,{httpOnly:true,sameSite:"lax",secure:process.env.NODE_ENV==="production" && request.nextUrl.protocol==="https:",path:"/",maxAge:15*60});
  return response;
}
export async function DELETE(request: NextRequest) {
  if (!validOrigin(request)) return NextResponse.json({error:"Invalid origin"}, {status:403});
  const token=request.cookies.get(cookieName)?.value;
  if (token) {
    const base=process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080";
    await fetch(base+"/v1/auth/logout",{method:"POST",headers:{Authorization:"Bearer "+token},signal:AbortSignal.timeout(5000)}).catch(()=>{});
  }
  const response=NextResponse.json({ok:true});
  response.cookies.delete(cookieName);
  return response;
}
