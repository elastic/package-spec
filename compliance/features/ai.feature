Feature: Support for AI-related features

  @3.4.0
  Scenario: Content package includes AI prompt
   Given the "good_content" package is installed
    Then there is a security AI prompt "good_content-security-ai-prompt-1"
