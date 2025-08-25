
<p align="right">
   <a href="./README.md">中文</a> | <strong>English</strong>
</p>


<p align="center">
   <picture>
   <img style="width: 80%" src="https://pic1.imgdb.cn/item/6846e33158cb8da5c83eb1eb.png" alt="image__3_-removebg-preview.png"> 
    </picture>
</p>

<div align="center">

_This project is a secondary development based on [one-hub](https://github.com/MartialBE/one-api)_

<a href="https://t.me/+LGKwlC_xa-E5ZDk9">
  <img src="https://img.shields.io/badge/Telegram-AI Wave Community-0088cc?style=for-the-badge&logo=telegram&logoColor=white" alt="Telegram Group" />
</a>

<sup><i>AI Wave Community</i></sup> · <sup><i>(Offering public API and AI bots in-group)</i></sup>

### [📚 View Original Project Documentation](https://one-hub-doc.vercel.app/)

</div>


## Current Differences from the Original Version (Latest Image)

- Supports **batch channel deletion**
- Supports `LinuxDo` login
- Supports **dark mode following system settings**
- Supports **batch adding models to multiple channels**
- Supports configuring **case-insensitive model names**
- Supports configuring **unified request-response model names**
- Supports **removing specified parameters** from channel extra parameters
- Supports **model variable substitution** in channel `BaseURL`
- Supports **pass-through of extra parameters** for native `/gemini` image generation requests
- Supports **custom channels** using native `Claude` routing - integrating `ClaudeCode`
- Supports `VertexAI` channels using native `Gemini` routing - integrating `GeminiCli`
- Supports `VertexAI` channels using native `Claude` routing - integrating `ClaudeCode`
- Supports configuring multiple `Regions` for `VertexAI` channels, with random `Region` selection per request
- Supports native **video generation requests** (`Veo` series models) for `Google Gemini` channels via `/gemini`
- Supports `gemini-2.0-flash-preview-image-generation` for text-to-image/image-to-image, compatible with `OpenAI` chat interfaces
- Added **invitation code settings** module
- Added **user group functionality for batch channel addition**
- Added **configuration for whether empty responses are billed** (default: billed)
- Added **time period conditions in analytics - top-up statistics**
- Added **RPM / TPM / CPM display in analytics**
- Added **invitation top-up rebate feature** (optional types: fixed/percentage)
- Fixed `bug` where user-related APIs were ineffective
- Fixed `bug` with missing fields in invitation records
- Fixed `bug` where hardcoded timezones affected statistical data
- Fixed `bug` with payment callback exceptions in multi-instance deployments
- Fixed `bug` with floating-point `token` calculations for Zhipu `GLM` models
- Fixed `bug` where API routing allowed `CDN` caching, leading to privilege escalation
- Fixed several `bugs` where user quota cache inconsistencies with `DB` data caused billing anomalies
- Removed meaningless original price-related styles from log functionality
- Optimized email rule validation
- Optimized several `UI` interactions
- Optimized logic for disabled channel email notifications
- Optimized `VertexAI` authentication caching
- ...

## Deployment

> Follow the original deployment tutorial and replace the image with `deanxv/done-hub`.

> Database-compatible; the original version can directly pull this image for migration.

## Acknowledgments

- This program uses the following open-source projects:
  - [one-hub](https://github.com/MartialBE/one-api) as the foundation of this project

Thanks to the authors and contributors of the above projects.