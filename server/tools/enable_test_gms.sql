-- GX distributed test accounts only. Run against the intended offline database.
-- GM authority uses Admin (0 normal, 1 GM), independently of AdminLevel.
UPDATE PS_UserData_Users_Master SET Admin=1
WHERE UserID IN ('admin1','admin2','admin3','admin4','admin5',
                 'normal1','normal2','normal3','normal4','normal5');
