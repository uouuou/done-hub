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
  <img src="https://img.shields.io/badge/Telegram-AI Wave Group-0088cc?style=for-the-badge&logo=telegram&logoColor=white" alt="Telegram Group" />
</a>

<sup><i>AI Wave Community</i></sup> · <sup><i>(Public API and AI bots available in the group)</i></sup>

### [📚 Click to view original project documentation](https://one-hub-doc.vercel.app/)

</div>


## Current differences from the original version (latest image)

- Support for **batch deletion of channels**
- Support for `LinuxDo` login
- Support for **night mode following system configuration**
- Support for **batch adding models to multiple channels**
- Support for configuring **case-insensitive model names**
- Support for configuring **request-response unified model names**
- Support for **removing specific parameters from channel extra parameters**
- Support for **model variable replacement** in channel `BaseURL`
- Support for **extra parameter passing** in native `/gemini` image generation requests
- Support for custom channels using `Claude` native routes - integrating `ClaudeCode`
- Support for `VertexAI` channels using `Gemini` native routes - integrating `GeminiCli`
- Support for `VertexAI` channels using `Claude` native routes - integrating `ClaudeCode`
- Support for configuring multiple `Regions` under `VertexAI` channels, randomly selecting a `Region` for each request
- Support for `gemini-2.0-flash-preview-image-generation` text-to-image/image-to-image, compatible with `OpenAI` conversation interface
- Added **user grouping function for batch adding channels**
- Added **configuration for whether empty replies are charged** (Default: charged)
- Added **time period conditions in recharge statistics in analysis function**
- Added **RPM / TPM / CPM display in analysis function**
- Added **invitation recharge rebate function** (Optional types: fixed/percentage)
- Fixed bug where user-related interfaces were ineffective
- Fixed bug with missing invitation record fields
- Fixed bug where hardcoded timezone affected statistical data
- Fixed bug with payment callback exceptions in multi-instance deployments
- Fixed bug with floating-point calculation of `token` in Zhipu `GLM` models
- Fixed bug where allowing `cdn` caching under `API` routes caused unauthorized access
- Fixed several bugs where user quota cache inconsistency with `DB` data caused billing exceptions
- Removed meaningless original price-related styles in log function
- Optimized email rule validation
- Optimized various `UI` interactions
- Optimized disabled channel email notification logic
- ...

## Deployment

> Follow the original deployment tutorial and replace the image with `deanxv/done-hub`.

> Database compatible, the original version can directly pull this image for migration.

## Acknowledgements

- This program uses the following open source projects
  - [one-hub](https://github.com/MartialBE/one-api) as the foundation for this project

Thanks to the authors and contributors of the above projects
