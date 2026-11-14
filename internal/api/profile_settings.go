package api

import (
 "net/http"
 "errors"
 "github.com/jackc/pgx/v5"
 "strings"
)

type settingsDTO struct {
 AddressLine1 string `json:"address_line1"`
 AddressLine2 string `json:"address_line2"`
 City string `json:"city"`
 PostalCode string `json:"postal_code"`
 ResidenceCountry string `json:"residence_country"`
 NotificationsEmail bool `json:"notifications_email"`
 NotificationsPush bool `json:"notifications_push"`
 NotificationsSecurity bool `json:"notifications_security"`
 Appearance string `json:"appearance"`
}

func (s *Server) getProfileSettings(w http.ResponseWriter,r *http.Request) {
 var v settingsDTO
 err:=s.db.QueryRow(r.Context(),`SELECT address_line1,address_line2,city,postal_code,COALESCE(residence_country,''),notifications_email,notifications_push,notifications_security,appearance FROM user_profile_settings WHERE user_id=$1`,uid(r)).Scan(&v.AddressLine1,&v.AddressLine2,&v.City,&v.PostalCode,&v.ResidenceCountry,&v.NotificationsEmail,&v.NotificationsPush,&v.NotificationsSecurity,&v.Appearance)
 if err!=nil {
  if !errors.Is(err,pgx.ErrNoRows) {writeError(w,500,"internal_error","Could not load settings");return}
  // New users inherit defaults without requiring a separate initialization path.
  if _,e:=s.db.Exec(r.Context(),`INSERT INTO user_profile_settings(user_id) VALUES($1) ON CONFLICT DO NOTHING`,uid(r));e!=nil {writeError(w,500,"internal_error","Could not initialize settings");return}
  err=s.db.QueryRow(r.Context(),`SELECT address_line1,address_line2,city,postal_code,COALESCE(residence_country,''),notifications_email,notifications_push,notifications_security,appearance FROM user_profile_settings WHERE user_id=$1`,uid(r)).Scan(&v.AddressLine1,&v.AddressLine2,&v.City,&v.PostalCode,&v.ResidenceCountry,&v.NotificationsEmail,&v.NotificationsPush,&v.NotificationsSecurity,&v.Appearance)
  if err!=nil {writeError(w,500,"internal_error","Could not load settings");return}
 }
 writeJSON(w,200,v)
}

type addressUpdate struct {
 AddressLine1 string `json:"address_line1"`
 AddressLine2 string `json:"address_line2"`
 City string `json:"city"`
 PostalCode string `json:"postal_code"`
}
func (s *Server) putAddress(w http.ResponseWriter,r *http.Request) {
 var v addressUpdate
 if !decode(w,r,&v){return}
 v.AddressLine1=strings.TrimSpace(v.AddressLine1)
 v.AddressLine2=strings.TrimSpace(v.AddressLine2)
 v.City=strings.TrimSpace(v.City)
 v.PostalCode=strings.TrimSpace(v.PostalCode)
 if len(v.AddressLine1)>200||len(v.AddressLine2)>200||len(v.City)>120||len(v.PostalCode)>32 {writeError(w,400,"validation_error","Address fields are too long");return}
 _,err:=s.db.Exec(r.Context(),`INSERT INTO user_profile_settings(user_id,address_line1,address_line2,city,postal_code) VALUES($1,$2,$3,$4,$5) ON CONFLICT(user_id) DO UPDATE SET address_line1=EXCLUDED.address_line1,address_line2=EXCLUDED.address_line2,city=EXCLUDED.city,postal_code=EXCLUDED.postal_code,updated_at=now()`,uid(r),v.AddressLine1,v.AddressLine2,v.City,v.PostalCode)
 if err!=nil {writeError(w,500,"internal_error","Could not save address");return}
 s.getProfileSettings(w,r)
}
type notificationUpdate struct {
 Email *bool `json:"email"`
 Push *bool `json:"push"`
 Security *bool `json:"security"`
}
func (s *Server) putNotifications(w http.ResponseWriter,r *http.Request) {
 var v notificationUpdate
 if !decode(w,r,&v){return}
 if v.Email==nil||v.Push==nil||v.Security==nil {writeError(w,400,"validation_error","All notification preferences are required");return}
 // Security notifications cannot be disabled through preferences.
 if !*v.Security {writeError(w,400,"security_required","Security notifications cannot be disabled");return}
 _,err:=s.db.Exec(r.Context(),`INSERT INTO user_profile_settings(user_id,notifications_email,notifications_push,notifications_security) VALUES($1,$2,$3,TRUE) ON CONFLICT(user_id) DO UPDATE SET notifications_email=EXCLUDED.notifications_email,notifications_push=EXCLUDED.notifications_push,notifications_security=TRUE,updated_at=now()`,uid(r),*v.Email,*v.Push)
 if err!=nil {writeError(w,500,"internal_error","Could not save notifications");return}
 s.getProfileSettings(w,r)
}
type appearanceUpdate struct { Appearance string `json:"appearance"` }
func (s *Server) putAppearance(w http.ResponseWriter,r *http.Request) {
 var v appearanceUpdate
 if !decode(w,r,&v){return}
 if v.Appearance!="system"&&v.Appearance!="light"&&v.Appearance!="dark" {writeError(w,400,"validation_error","Invalid appearance");return}
 _,err:=s.db.Exec(r.Context(),`INSERT INTO user_profile_settings(user_id,appearance) VALUES($1,$2) ON CONFLICT(user_id) DO UPDATE SET appearance=EXCLUDED.appearance,updated_at=now()`,uid(r),v.Appearance)
 if err!=nil {writeError(w,500,"internal_error","Could not save appearance");return}
 s.getProfileSettings(w,r)
}
func (s *Server) legalIdentityChangeStub(w http.ResponseWriter,r *http.Request) {
 writeError(w,http.StatusNotImplemented,"veriff_not_configured","Legal identity and residence country changes require Veriff re-verification; this integration is not configured")
}
