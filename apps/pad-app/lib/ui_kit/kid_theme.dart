import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

class KidColors {
  // Strictly Limited Palette
  static const Color color1 = Color(0xFF025373); // Deep Teal
  static const Color color2 = Color(0xFF0388A6); // Azure Blue
  static const Color color3 = Color(0xFF038C3E); // Forest Green (Success)
  static const Color color4 = Color(0xFFF2A516); // Sunny Yellow
  static const Color color5 = Color(0xFFF27830); // Coral Orange
  
  static const Color black = Color(0xFF000000);
  static const Color white = Color(0xFFFFFFFF);
  static const Color background = Color(0xFFFFFFFF); // Keep it clean white
  static const Color border = Color(0xFF000000);     // High contrast black border
}

class KidTheme {
  static ThemeData get light {
    final base = ThemeData.light();
    return base.copyWith(
      scaffoldBackgroundColor: KidColors.background,
      textTheme: GoogleFonts.nunitoTextTheme(base.textTheme).apply(
        bodyColor: KidColors.black,
        displayColor: KidColors.black,
      ),
      colorScheme: base.colorScheme.copyWith(
        primary: KidColors.color1,
        secondary: KidColors.color3,
        surface: KidColors.white,
      ),
      appBarTheme: base.appBarTheme.copyWith(
        backgroundColor: KidColors.white,
        elevation: 0,
        centerTitle: true,
        titleTextStyle: GoogleFonts.nunito(
          color: KidColors.black,
          fontSize: 22,
          fontWeight: FontWeight.w900,
          letterSpacing: -0.5,
        ),
        iconTheme: const IconThemeData(color: KidColors.black),
      ),
    );
  }
}
