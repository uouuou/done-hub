
<p align="right">
   <a href="./README.md">中文</a> | <strong>English</strong>
</p>

<p align="center">
   <picture>
   <img style="width: 80%" src="https://pic1.imgdb.cn/item/6846e33158cb8da5c83eb1eb.png" alt="Project Logo"> 
    </picture>
</p>

<div align="center">

_This project is an enhanced fork of the foundational [one-hub](https://github.com/MartialBE/one-api) framework._

<a href="https://t.me/+LGKwlC_xa-E5ZDk9">
  <img src="https://img.shields.io/badge/Telegram-AI%20Wave%20Community-0088cc?style=for-the-badge&logo=telegram&logoColor=white" alt="Telegram Community" />
</a>

<sup><i>The AI Wave Community</i></sup> · <sup><i>(Public APIs and AI assistants available)</i></sup>

### [📚 View the Original Project Documentation](https://one-hub-doc.vercel.app/)

</div>

## Key Enhancements and Modifications

This version introduces significant upgrades and stability improvements over the latest official build. The key distinctions are as follows:

### New Features & Core Enhancements
- **Channel Management:** Introduced support for bulk deletion of channels.
- **Authentication:** Enabled user authentication via `LinuxDo`.
- **User Interface:** Implemented system-adaptive dark mode.
- **Model Management:** Added functionality for batch-adding models to multiple channels.
- **Configuration:** Enabled case-insensitive configuration for model names.
- **API Consistency:** Introduced an option to unify model names across requests and responses.
- **Parameter Control:** Allowed for the removal of specific parameters within a channel's additional settings.
- **Dynamic Routing:** Implemented model variable substitution within channel `BaseURL` configurations.
- **Gemini Native Support:** Enabled pass-through of additional parameters for native Gemini image generation requests.
- **Expanded Claude Support:** Integrated native Claude routing for custom channels via `ClaudeCode`.
- **VertexAI Integration:**
  - Enabled native Gemini routing via `GeminiCli`.
  - Enabled native Claude routing via `ClaudeCode`.
  - Implemented multi-region support, with random region selection per request to enhance availability.
- **Google Gemini Video Generation:** Added support for native video generation requests (`Veo` series models).
- **Advanced Image Generation:** Integrated support for `gemini-2.0-flash-preview-image-generation` (text-to-image and image-to-image), ensuring compatibility with the OpenAI chat interface.
- **User Grouping:** Implemented a feature for assigning user groups to channels in bulk.
- **Billing Control:** Added a configuration to determine if empty API responses incur charges (Default: Enabled).
- **Analytics:**
  - Added time period filters to the deposit statistics report.
  - Incorporated RPM (Requests Per Minute), TPM (Tokens Per Minute), and CPM (Calls Per Minute) metrics into the analytics dashboard.
- **Referral Program:** Deployed a new referral system with commission rewards (configurable as fixed amount or percentage).

### Bug Fixes & Optimizations
- **API Integrity:** Resolved critical bugs affecting user-related API endpoints.
- **Data Integrity:** Corrected an issue of missing fields in referral records.
- **System Stability:** Addressed a bug where hardcoded timezones skewed statistical data.
- **Payment Processing:** Rectified a payment callback anomaly that occurred in multi-instance deployments.
- **Billing Accuracy:**
  - Fixed a floating-point calculation error for Zhipu `GLM` model tokenization.
  - Resolved multiple bugs causing discrepancies between cached user quotas and database records, which led to billing inaccuracies.
- **Security:** Patched a privilege escalation vulnerability caused by improper CDN caching on API routes.
- **User Experience:**
  - Removed extraneous styling related to original pricing from the log interface.
  - Enhanced email validation logic.
  - Refined several UI interactions for improved clarity and usability.
- **System Performance:**
  - Optimized the email notification logic for disabled channels.
  - Improved the authentication caching mechanism for `VertexAI`.
- ...and numerous other minor enhancements.

## Deployment

> To deploy, follow the original project's installation guide, substituting the Docker image with `deanxv/done-hub`.

> This project maintains full database compatibility, allowing for a seamless migration from the original version by simply updating the image.

## Acknowledgements

This initiative is built upon the contributions of the following open-source projects:
- **[one-hub](https://github.com/MartialBE/one-api):** The foundational framework upon which this project is based.

We extend our sincere gratitude to the authors and contributors of these foundational projects.