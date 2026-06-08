#[cfg(target_os = "windows")]
use std::os::windows::process::CommandExt;
use std::process::Command;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
  tauri::Builder::default()
    .setup(|app| {
      #[cfg(target_os = "windows")]
      {
        // Kích hoạt WSL và đợi lệnh này thực thi xong
        let _ = Command::new("wsl")
          .arg("--exec")
          .arg("true")
          .creation_flags(0x08000000)
          .status();

        // Tạm dừng 3 giây để WSL kịp khởi động Docker và các dịch vụ nền
        std::thread::sleep(std::time::Duration::from_secs(3));
      }


      if cfg!(debug_assertions) {
        app.handle().plugin(
          tauri_plugin_log::Builder::default()
            .level(log::LevelFilter::Info)
            .build(),
        )?;
      }
      Ok(())
    })
    .build(tauri::generate_context!())
    .expect("error while building tauri application")
    .run(|_app_handle, event| {
      if let tauri::RunEvent::ExitRequested { .. } = event {
        #[cfg(target_os = "windows")]
        {
          // Khi đóng app, tắt WSL để giải phóng RAM (vì đây là app thay thế Docker Desktop)
          let _ = Command::new("wsl")
            .arg("--shutdown")
            .creation_flags(0x08000000)
            .spawn();
        }
      }
    });
}


