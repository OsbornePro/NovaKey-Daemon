<#
.SYNOPSIS
This cmdlet is used to impersonate what a phone app or other device would submit to have the daemon type a secret


.DESCRIPTION
Impersonate what a phone app or other device would submit to have the daemon type a secret


.PARAMETER Address
Define the address of the NovaKey-Daemon server

.PARAMETER DeviceID
Define the ID of the device

.PARAMETER Password
Define the password to type

.PARAMETER KeyHex
Define the device secret

.PARAMETER ServerKyberPubBase64
Define the base64 value of the public Kyber certificate
#>
[CmdletBinding()]
    param(
        [Parameter(
            Mandatory=$False
        )]  # End Parameter
        [String]$Address = '127.0.0.1:60768',

        [Parameter(
            Mandatory=$False
        )]  # End Parameter
        [String]$DeviceID = "phone",

        [Parameter(
            Mandatory=$False
        )]  # End Parameter
        [String]$Password ="SuperStrongPassword123!",

        [Parameter(
            Mandatory=$False
        )]  # End Parameter
        [String]$KeyHex = "7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234",

        [Parameter(
            Mandatory=$False
        )]  # End Parameter
        [String]$ServerKyberPubBase64 ="wgkdGfeAyuqswle8f9e9Aagxc1gQr8ZZhpu2OrIlLZIadRlxMPRWEiWCf+YVfhlsVsHIb9amBboFxuOJxpwXVCiUzEmhW1GSvRAtRGk5NGRQLMdqLhTIrCcY//JKThWJauwjJmC0+6CwhUvHUNpzhQdYRsLMdcN7PxaNu1YfZ+se/5kYmuUfvTQYdUycmGGTI8KUGqFXZAasXgYO9eS/QjchYfg/PBorutxsXempesafyrQly/k4/OiosyzOLfYqMsKjH0c8ftXJCEZ/hSuZkFU7ZQSIadMXSPQkgGk5fiJEmJAoNhvDbhmNdRddpRcGX6obxCOjREpl8pxkA1XF1/SMw9dtddwRPGrPh8oUmBEd9gltDzqMa4yxz/hrj8NF6NIWXHW7GZyZIjwetUt8jBilTZeqIcMwkwtyOHyRzwxyQHx5ncgg2lpZYDKIXlEGO5wHfhq2QbDGkhxXj7LPpuKDHUdPe3TEN9cgrjeMHSZx60jEWxuqiYwGNaSr0WgmKBCFGEK5xblimnqJTXpQDZdyVyo0M7RVPdtTKxtmwAnNxFqGVNd0bWYcuji3ACOKb9wVi2VaaPeQDrYYrqpYpJqHhIZtNJKJ+zwEPDMDhfQ+q7gkd9cR4/INkhSTyhsnmhaipdAZRqBjWoVRcosHaYFbpGVFO+KA9iFDEJSA38EYyPVDsLk7WovBvAsmFoJEUcOZTEOuMoWqn+oYTMAAyXUmtXxCzPx2P4tq31NMotKnAIobfdA7x2x60YEwupssMXRzXdlCStcHHPCdelwKBicO/4agVXl1PKFyYVEl2SoA1Ss+oqwVVWanS3MrqvVSVeg4AHN7KcUDseVgm0UAGCiONoByviYT5Tdg9eeCRYS0m0xQT5uTlRWHO3kriquLniBcS2FBGdtnwhnIwDtd2ptsnpCsVJkFGTJckcIPNdBRgMG0nOSmXaU3ttsSCZtrg2RVMdApjTVOuWQaocgDeLI9iFe7fOQYM3OHu/iErkldXeVaJKiOfeu8inVG3/gJl3pJ/DcCjuuVF6NQjwuJ6BE3OSsYkeqs1vutEaOmsfAPjATPhSMWwLu+RqK/FqetooyqeKgPejRLWTJ+6HSr3eDPeax/J1ZdCMYwd3sDTFdAJXctDVY9d8KcgfIET0ehAPK2d9Fo6fqH98V+4WJ77bEJ+PMVGPAp1ApVuLg966xSzRh6Zjw4VyA+ItZpTiZPyXiPgKmDFurLhQImMNIJC9XJcgAZvlsC7fdPhnRvNduD3TueE8sEpMkt5UuZQcvJUbY7Etq3IgU3YPOpaIlF+9nME3gorxeEI0ePPjZWVlwihRkPJuEjXDQg3fxxPia64FZaOAkMw6RVpucQ51MzBSJBybe8dPum8Cdi7bpyDdKnvvDI3TqPFhYmLSo+MIpVHtrNvQeryQmax+CN6WOGlDRftKApkrwUvkGmdkte6fdeEEFQH2RlMmRmJiJIHKMoFCw2iKQAclqZfHMKEyFWE8wWceQV2LvHeQEy3RbM21QALCqTYdkrn4dcx5QX/tssHIglSWeutK1hTKrbAn2yyVlmSfVxKfqiE48cYQSXLbk="
    )  # End param

Write-Warning -Message "Click into the browser address bar or somewhere to test the typing that will happen"
Start-Sleep -Seconds 3
If (!(Test-Path -Path .\dist\nvclient.exe)) {
    go build -o .\dist\nvclient.exe .\cmd\nvclient
}  # End If
.\dist\nvclient.exe -addr $Address -device-id $DeviceID -password $Password -key-hex $KeyHex -server-kyber-pub-b64 $ServerKyberPubBase64
