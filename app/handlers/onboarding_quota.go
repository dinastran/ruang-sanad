package handlers

import (
 "strconv"

 "github.com/gofiber/fiber/v2"
 "github.com/maulanashalihin/laju-go/app/services"
)

func (h *OnboardingHandler) ClassControls(c *fiber.Ctx) error {
 sess,_:=h.store.Get(c)
 user:=sessionUser(sess)
 board,err:=h.service.ClassControls(c.UserContext())
 if err!=nil{return c.Status(fiber.StatusInternalServerError).SendString("Gagal memuat kontrol kuota")}
 return h.inertia.Render(c,"app/OnboardingQuota",fiber.Map{"user":user,"board":board})
}

func onboardingClassID(c *fiber.Ctx) (int64,error) {
 id,err:=strconv.ParseInt(c.Params("id"),10,64)
 if err!=nil||id<=0 {return 0,services.ErrOnboardingNotFound}
 return id,nil
}
func (h *OnboardingHandler) ClassQuota(c *fiber.Ctx) error {
 id,err:=onboardingClassID(c)
 if err!=nil{return onboardingHTTPError(c,err)}
 var req struct {Target int64 `json:"target"`}
 if err:=c.BodyParser(&req);err!=nil{return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"Target tidak valid"})}
 if err:=h.service.SetClassQuota(c.UserContext(),id,c.Locals("user_id").(int64),req.Target);err!=nil{
  return onboardingHTTPError(c,err)
 }
 return c.JSON(fiber.Map{"ok":true})
}
func (h *OnboardingHandler) ClassAction(c *fiber.Ctx) error {
 id,err:=onboardingClassID(c)
 if err!=nil{return onboardingHTTPError(c,err)}
 var req struct {
  Action string `json:"action"`
  Note string `json:"note"`
 }
 if err:=c.BodyParser(&req);err!=nil{return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"Tindakan tidak valid"})}
 if err:=h.service.ClassAction(c.UserContext(),id,c.Locals("user_id").(int64),req.Action,req.Note);err!=nil{
  return onboardingHTTPError(c,err)
 }
 return c.JSON(fiber.Map{"ok":true})
}
func (h *OnboardingHandler) ClassHistory(c *fiber.Ctx) error {
 id,err:=onboardingClassID(c)
 if err!=nil{return onboardingHTTPError(c,err)}
 rows,err:=h.service.ClassActions(c.UserContext(),id)
 if err!=nil{return onboardingHTTPError(c,err)}
 return c.JSON(fiber.Map{"items":rows})
}
