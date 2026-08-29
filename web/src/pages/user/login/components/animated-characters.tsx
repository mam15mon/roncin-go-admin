import React, { memo, useCallback, useEffect, useRef, useState } from 'react';

const AUTH_CHARACTER_COLORS = {
  eye: '#ffffff',
  pupil: '#262b33',
  purple: '#7c3aed',
  charcoal: '#262b33',
  orange: '#ff826b',
  yellow: '#edd953',
} as const;

interface Position {
  x: number;
  y: number;
}

interface PupilProps {
  size?: number;
  maxDistance?: number;
  pupilColor?: string;
  forceLookX?: number;
  forceLookY?: number;
}

export const Pupil = memo(
  ({
    size = 12,
    maxDistance = 5,
    pupilColor = AUTH_CHARACTER_COLORS.pupil,
    forceLookX,
    forceLookY,
  }: PupilProps) => {
    const [pupilPosition, setPupilPosition] = useState<Position>({ x: 0, y: 0 });
    const pupilRef = useRef<HTMLDivElement>(null);

    const calculatePosition = useCallback(
      (mx: number, my: number) => {
        if (!pupilRef.current) return { x: 0, y: 0 };
        if (forceLookX !== undefined && forceLookY !== undefined) {
          return { x: forceLookX, y: forceLookY };
        }

        const pupil = pupilRef.current.getBoundingClientRect();
        const pupilCenterX = pupil.left + pupil.width / 2;
        const pupilCenterY = pupil.top + pupil.height / 2;

        const deltaX = mx - pupilCenterX;
        const deltaY = my - pupilCenterY;
        const distance = Math.min(Math.sqrt(deltaX ** 2 + deltaY ** 2), maxDistance);

        const angle = Math.atan2(deltaY, deltaX);
        return {
          x: Math.cos(angle) * distance,
          y: Math.sin(angle) * distance,
        };
      },
      [forceLookX, forceLookY, maxDistance],
    );

    useEffect(() => {
      const handleMouseMove = (e: MouseEvent) => {
        setPupilPosition(calculatePosition(e.clientX, e.clientY));
      };

      window.addEventListener('mousemove', handleMouseMove);
      return () => window.removeEventListener('mousemove', handleMouseMove);
    }, [calculatePosition]);

    const renderedPupilPosition =
      forceLookX !== undefined && forceLookY !== undefined
        ? { x: forceLookX, y: forceLookY }
        : pupilPosition;

    return (
      <div
        ref={pupilRef}
        style={{
          width: `${size}px`,
          height: `${size}px`,
          borderRadius: '9999px',
          backgroundColor: pupilColor,
          transform: `translate(${renderedPupilPosition.x}px, ${renderedPupilPosition.y}px)`,
          transition: 'transform 0.1s ease-out',
        }}
      />
    );
  },
);
Pupil.displayName = 'Pupil';

interface EyeBallProps {
  size?: number;
  pupilSize?: number;
  maxDistance?: number;
  eyeColor?: string;
  pupilColor?: string;
  isBlinking?: boolean;
  forceLookX?: number;
  forceLookY?: number;
}

export const EyeBall = memo(
  ({
    size = 48,
    pupilSize = 16,
    maxDistance = 10,
    eyeColor = AUTH_CHARACTER_COLORS.eye,
    pupilColor = AUTH_CHARACTER_COLORS.pupil,
    isBlinking = false,
    forceLookX,
    forceLookY,
  }: EyeBallProps) => {
    const [pupilPosition, setPupilPosition] = useState<Position>({ x: 0, y: 0 });
    const eyeRef = useRef<HTMLDivElement>(null);

    const calculatePosition = useCallback(
      (mx: number, my: number) => {
        if (!eyeRef.current) return { x: 0, y: 0 };
        if (forceLookX !== undefined && forceLookY !== undefined) {
          return { x: forceLookX, y: forceLookY };
        }

        const eye = eyeRef.current.getBoundingClientRect();
        const eyeCenterX = eye.left + eye.width / 2;
        const eyeCenterY = eye.top + eye.height / 2;

        const deltaX = mx - eyeCenterX;
        const deltaY = my - eyeCenterY;
        const distance = Math.min(Math.sqrt(deltaX ** 2 + deltaY ** 2), maxDistance);

        const angle = Math.atan2(deltaY, deltaX);
        return {
          x: Math.cos(angle) * distance,
          y: Math.sin(angle) * distance,
        };
      },
      [forceLookX, forceLookY, maxDistance],
    );

    useEffect(() => {
      const handleMouseMove = (e: MouseEvent) => {
        setPupilPosition(calculatePosition(e.clientX, e.clientY));
      };

      window.addEventListener('mousemove', handleMouseMove);
      return () => window.removeEventListener('mousemove', handleMouseMove);
    }, [calculatePosition]);

    const renderedPupilPosition =
      forceLookX !== undefined && forceLookY !== undefined
        ? { x: forceLookX, y: forceLookY }
        : pupilPosition;

    return (
      <div
        ref={eyeRef}
        style={{
          width: `${size}px`,
          height: isBlinking ? '2px' : `${size}px`,
          borderRadius: '9999px',
          backgroundColor: eyeColor,
          overflow: 'hidden',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          transition: 'height 0.15s ease, transform 0.15s ease',
        }}
      >
        {!isBlinking && (
          <div
            style={{
              width: `${pupilSize}px`,
              height: `${pupilSize}px`,
              borderRadius: '9999px',
              backgroundColor: pupilColor,
              transform: `translate(${renderedPupilPosition.x}px, ${renderedPupilPosition.y}px)`,
              transition: 'transform 0.1s ease-out',
            }}
          />
        )}
      </div>
    );
  },
);
EyeBall.displayName = 'EyeBall';

interface CharacterState {
  faceX: number;
  faceY: number;
  bodySkew: number;
}

export interface AnimatedCharactersProps {
  isTyping?: boolean;
  showPassword?: boolean;
  passwordLength?: number;
}

export const AnimatedCharacters = memo(
  ({
    isTyping = false,
    showPassword = false,
    passwordLength = 0,
  }: AnimatedCharactersProps) => {
    const [isPurpleBlinking, setIsPurpleBlinking] = useState(false);
    const [isBlackBlinking, setIsBlackBlinking] = useState(false);
    const [isLookingAtEachOther, setIsLookingAtEachOther] = useState(false);
    const [isPurplePeeking, setIsPurplePeeking] = useState(false);

    const purpleRef = useRef<HTMLDivElement>(null);
    const blackRef = useRef<HTMLDivElement>(null);
    const yellowRef = useRef<HTMLDivElement>(null);
    const orangeRef = useRef<HTMLDivElement>(null);

    const [purplePos, setPurplePos] = useState<CharacterState>({ faceX: 0, faceY: 0, bodySkew: 0 });
    const [blackPos, setBlackPos] = useState<CharacterState>({ faceX: 0, faceY: 0, bodySkew: 0 });
    const [yellowPos, setYellowPos] = useState<CharacterState>({ faceX: 0, faceY: 0, bodySkew: 0 });
    const [orangePos, setOrangePos] = useState<CharacterState>({ faceX: 0, faceY: 0, bodySkew: 0 });

    const calculatePosForRef = useCallback(
      (ref: React.RefObject<HTMLDivElement | null>, mx: number, my: number) => {
        if (!ref.current) return { faceX: 0, faceY: 0, bodySkew: 0 };

        const rect = ref.current.getBoundingClientRect();
        const centerX = rect.left + rect.width / 2;
        const centerY = rect.top + rect.height / 3;

        const deltaX = mx - centerX;
        const deltaY = my - centerY;

        return {
          faceX: Math.max(-15, Math.min(15, deltaX / 20)),
          faceY: Math.max(-10, Math.min(10, deltaY / 30)),
          bodySkew: Math.max(-6, Math.min(6, -deltaX / 120)),
        };
      },
      [],
    );

    useEffect(() => {
      const handleMouseMove = (e: MouseEvent) => {
        setPurplePos(calculatePosForRef(purpleRef, e.clientX, e.clientY));
        setBlackPos(calculatePosForRef(blackRef, e.clientX, e.clientY));
        setYellowPos(calculatePosForRef(yellowRef, e.clientX, e.clientY));
        setOrangePos(calculatePosForRef(orangeRef, e.clientX, e.clientY));
      };

      window.addEventListener('mousemove', handleMouseMove);
      return () => window.removeEventListener('mousemove', handleMouseMove);
    }, [calculatePosForRef]);

    // Blinking effect for purple character
    useEffect(() => {
      let blinkTimer: ReturnType<typeof setTimeout> | undefined;
      let resetTimer: ReturnType<typeof setTimeout> | undefined;
      let cancelled = false;
      const getRandomBlinkInterval = () => Math.random() * 4000 + 3000;
      const scheduleBlink = () => {
        blinkTimer = setTimeout(() => {
          if (cancelled) return;
          setIsPurpleBlinking(true);
          resetTimer = setTimeout(() => {
            if (cancelled) return;
            setIsPurpleBlinking(false);
            scheduleBlink();
          }, 150);
        }, getRandomBlinkInterval());
      };
      scheduleBlink();
      return () => {
        cancelled = true;
        if (blinkTimer) clearTimeout(blinkTimer);
        if (resetTimer) clearTimeout(resetTimer);
      };
    }, []);

    // Blinking effect for black character
    useEffect(() => {
      let blinkTimer: ReturnType<typeof setTimeout> | undefined;
      let resetTimer: ReturnType<typeof setTimeout> | undefined;
      let cancelled = false;
      const getRandomBlinkInterval = () => Math.random() * 4000 + 3000;
      const scheduleBlink = () => {
        blinkTimer = setTimeout(() => {
          if (cancelled) return;
          setIsBlackBlinking(true);
          resetTimer = setTimeout(() => {
            if (cancelled) return;
            setIsBlackBlinking(false);
            scheduleBlink();
          }, 150);
        }, getRandomBlinkInterval());
      };
      scheduleBlink();
      return () => {
        cancelled = true;
        if (blinkTimer) clearTimeout(blinkTimer);
        if (resetTimer) clearTimeout(resetTimer);
      };
    }, []);

    // Looking at each other animation when typing starts
    useEffect(() => {
      let startTimer: ReturnType<typeof setTimeout> | undefined;
      let stopTimer: ReturnType<typeof setTimeout> | undefined;

      if (isTyping) {
        startTimer = setTimeout(() => {
          setIsLookingAtEachOther(true);
        }, 0);
        stopTimer = setTimeout(() => {
          setIsLookingAtEachOther(false);
        }, 800);
      }

      return () => {
        if (startTimer) clearTimeout(startTimer);
        if (stopTimer) clearTimeout(stopTimer);
      };
    }, [isTyping]);

    // Purple sneaky peeking animation
    useEffect(() => {
      if (passwordLength <= 0 || !showPassword) {
        return;
      }

      let peekResetTimer: ReturnType<typeof setTimeout> | undefined;
      const firstPeek = setTimeout(() => {
        setIsPurplePeeking(true);
        peekResetTimer = setTimeout(() => {
          setIsPurplePeeking(false);
        }, 800);
      }, Math.random() * 3000 + 2000);

      return () => {
        clearTimeout(firstPeek);
        if (peekResetTimer) clearTimeout(peekResetTimer);
      };
    }, [passwordLength, showPassword]);

    const isHidingPassword = passwordLength > 0 && !showPassword;
    const isLookingAtEachOtherActive = isTyping && isLookingAtEachOther;
    const isPurplePeekingActive = passwordLength > 0 && showPassword && isPurplePeeking;

    return (
      <div
        style={{
          position: 'relative',
          width: '550px',
          height: '400px',
          userSelect: 'none',
        }}
      >
        {/* Purple tall rectangle character */}
        <div
          ref={purpleRef}
          style={{
            position: 'absolute',
            bottom: 0,
            left: '70px',
            width: '180px',
            height: isTyping || isHidingPassword ? '440px' : '400px',
            backgroundColor: AUTH_CHARACTER_COLORS.purple,
            borderRadius: '10px 10px 0 0',
            zIndex: 1,
            transform:
              passwordLength > 0 && showPassword
                ? 'skewX(0deg)'
                : isTyping || isHidingPassword
                  ? `skewX(${(purplePos.bodySkew || 0) - 12}deg) translateX(40px)`
                  : `skewX(${purplePos.bodySkew || 0}deg)`,
            transformOrigin: 'bottom center',
            transition: 'all 0.7s cubic-bezier(0.34, 1.56, 0.64, 1)',
          }}
        >
          <div
            style={{
              position: 'absolute',
              display: 'flex',
              gap: '32px',
              left:
                passwordLength > 0 && showPassword
                  ? '20px'
                  : isLookingAtEachOtherActive
                    ? '55px'
                    : `${45 + purplePos.faceX}px`,
              top:
                passwordLength > 0 && showPassword
                  ? '35px'
                  : isLookingAtEachOtherActive
                    ? '65px'
                    : `${40 + purplePos.faceY}px`,
              transition: 'all 0.7s cubic-bezier(0.34, 1.56, 0.64, 1)',
            }}
          >
            <EyeBall
              size={18}
              pupilSize={7}
              maxDistance={5}
              eyeColor={AUTH_CHARACTER_COLORS.eye}
              pupilColor={AUTH_CHARACTER_COLORS.pupil}
              isBlinking={isPurpleBlinking}
              forceLookX={
                passwordLength > 0 && showPassword
                  ? isPurplePeekingActive
                    ? 4
                    : -4
                  : isLookingAtEachOtherActive
                    ? 3
                    : undefined
              }
              forceLookY={
                passwordLength > 0 && showPassword
                  ? isPurplePeekingActive
                    ? 5
                    : -4
                  : isLookingAtEachOtherActive
                    ? 4
                    : undefined
              }
            />
            <EyeBall
              size={18}
              pupilSize={7}
              maxDistance={5}
              eyeColor={AUTH_CHARACTER_COLORS.eye}
              pupilColor={AUTH_CHARACTER_COLORS.pupil}
              isBlinking={isPurpleBlinking}
              forceLookX={
                passwordLength > 0 && showPassword
                  ? isPurplePeekingActive
                    ? 4
                    : -4
                  : isLookingAtEachOtherActive
                    ? 3
                    : undefined
              }
              forceLookY={
                passwordLength > 0 && showPassword
                  ? isPurplePeekingActive
                    ? 5
                    : -4
                  : isLookingAtEachOtherActive
                    ? 4
                    : undefined
              }
            />
          </div>
        </div>

        {/* Charcoal tall rectangle character */}
        <div
          ref={blackRef}
          style={{
            position: 'absolute',
            bottom: 0,
            left: '240px',
            width: '120px',
            height: '310px',
            backgroundColor: AUTH_CHARACTER_COLORS.charcoal,
            borderRadius: '8px 8px 0 0',
            zIndex: 2,
            transform:
              passwordLength > 0 && showPassword
                ? 'skewX(0deg)'
                : isLookingAtEachOtherActive
                  ? `skewX(${(blackPos.bodySkew || 0) * 1.5 + 10}deg) translateX(20px)`
                  : isTyping || isHidingPassword
                    ? `skewX(${(blackPos.bodySkew || 0) * 1.5}deg)`
                    : `skewX(${blackPos.bodySkew || 0}deg)`,
            transformOrigin: 'bottom center',
            transition: 'all 0.7s cubic-bezier(0.34, 1.56, 0.64, 1)',
          }}
        >
          <div
            style={{
              position: 'absolute',
              display: 'flex',
              gap: '24px',
              left:
                passwordLength > 0 && showPassword
                  ? '10px'
                  : isLookingAtEachOtherActive
                    ? '32px'
                    : `${26 + blackPos.faceX}px`,
              top:
                passwordLength > 0 && showPassword
                  ? '28px'
                  : isLookingAtEachOtherActive
                    ? '12px'
                    : `${32 + blackPos.faceY}px`,
              transition: 'all 0.7s cubic-bezier(0.34, 1.56, 0.64, 1)',
            }}
          >
            <EyeBall
              size={16}
              pupilSize={6}
              maxDistance={4}
              eyeColor={AUTH_CHARACTER_COLORS.eye}
              pupilColor={AUTH_CHARACTER_COLORS.pupil}
              isBlinking={isBlackBlinking}
              forceLookX={
                passwordLength > 0 && showPassword
                  ? -4
                  : isLookingAtEachOtherActive
                    ? 0
                    : undefined
              }
              forceLookY={
                passwordLength > 0 && showPassword
                  ? -4
                  : isLookingAtEachOtherActive
                    ? -4
                    : undefined
              }
            />
            <EyeBall
              size={16}
              pupilSize={6}
              maxDistance={4}
              eyeColor={AUTH_CHARACTER_COLORS.eye}
              pupilColor={AUTH_CHARACTER_COLORS.pupil}
              isBlinking={isBlackBlinking}
              forceLookX={
                passwordLength > 0 && showPassword
                  ? -4
                  : isLookingAtEachOtherActive
                    ? 0
                    : undefined
              }
              forceLookY={
                passwordLength > 0 && showPassword
                  ? -4
                  : isLookingAtEachOtherActive
                    ? -4
                    : undefined
              }
            />
          </div>
        </div>

        {/* Orange semi-circle character */}
        <div
          ref={orangeRef}
          style={{
            position: 'absolute',
            bottom: 0,
            left: '0px',
            width: '240px',
            height: '200px',
            zIndex: 3,
            backgroundColor: AUTH_CHARACTER_COLORS.orange,
            borderRadius: '120px 120px 0 0',
            transform:
              passwordLength > 0 && showPassword
                ? 'skewX(0deg)'
                : `skewX(${orangePos.bodySkew || 0}deg)`,
            transformOrigin: 'bottom center',
            transition: 'all 0.7s cubic-bezier(0.34, 1.56, 0.64, 1)',
          }}
        >
          <div
            style={{
              position: 'absolute',
              display: 'flex',
              gap: '32px',
              left:
                passwordLength > 0 && showPassword
                  ? '50px'
                  : `${82 + (orangePos.faceX || 0)}px`,
              top:
                passwordLength > 0 && showPassword
                  ? '85px'
                  : `${90 + (orangePos.faceY || 0)}px`,
              transition: 'all 0.2s ease-out',
            }}
          >
            <Pupil
              size={12}
              maxDistance={5}
              pupilColor={AUTH_CHARACTER_COLORS.pupil}
              forceLookX={passwordLength > 0 && showPassword ? -5 : undefined}
              forceLookY={passwordLength > 0 && showPassword ? -4 : undefined}
            />
            <Pupil
              size={12}
              maxDistance={5}
              pupilColor={AUTH_CHARACTER_COLORS.pupil}
              forceLookX={passwordLength > 0 && showPassword ? -5 : undefined}
              forceLookY={passwordLength > 0 && showPassword ? -4 : undefined}
            />
          </div>
        </div>

        {/* Yellow tall capsule character */}
        <div
          ref={yellowRef}
          style={{
            position: 'absolute',
            bottom: 0,
            left: '310px',
            width: '140px',
            height: '230px',
            backgroundColor: AUTH_CHARACTER_COLORS.yellow,
            borderRadius: '70px 70px 0 0',
            zIndex: 4,
            transform:
              passwordLength > 0 && showPassword
                ? 'skewX(0deg)'
                : `skewX(${yellowPos.bodySkew || 0}deg)`,
            transformOrigin: 'bottom center',
            transition: 'all 0.7s cubic-bezier(0.34, 1.56, 0.64, 1)',
          }}
        >
          <div
            style={{
              position: 'absolute',
              display: 'flex',
              gap: '24px',
              left:
                passwordLength > 0 && showPassword
                  ? '20px'
                  : `${52 + (yellowPos.faceX || 0)}px`,
              top:
                passwordLength > 0 && showPassword
                  ? '35px'
                  : `${40 + (yellowPos.faceY || 0)}px`,
              transition: 'all 0.2s ease-out',
            }}
          >
            <Pupil
              size={12}
              maxDistance={5}
              pupilColor={AUTH_CHARACTER_COLORS.pupil}
              forceLookX={passwordLength > 0 && showPassword ? -5 : undefined}
              forceLookY={passwordLength > 0 && showPassword ? -4 : undefined}
            />
            <Pupil
              size={12}
              maxDistance={5}
              pupilColor={AUTH_CHARACTER_COLORS.pupil}
              forceLookX={passwordLength > 0 && showPassword ? -5 : undefined}
              forceLookY={passwordLength > 0 && showPassword ? -4 : undefined}
            />
          </div>
          <div
            style={{
              position: 'absolute',
              height: '4px',
              width: '80px',
              borderRadius: '9999px',
              backgroundColor: AUTH_CHARACTER_COLORS.pupil,
              left:
                passwordLength > 0 && showPassword
                  ? '10px'
                  : `${40 + (yellowPos.faceX || 0)}px`,
              top:
                passwordLength > 0 && showPassword
                  ? '88px'
                  : `${88 + (yellowPos.faceY || 0)}px`,
              transition: 'all 0.2s ease-out',
            }}
          />
        </div>
      </div>
    );
  },
);
AnimatedCharacters.displayName = 'AnimatedCharacters';
