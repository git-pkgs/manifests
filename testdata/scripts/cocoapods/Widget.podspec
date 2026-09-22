Pod::Spec.new do |s|
  s.name = 'Widget'
  s.dependency 'Helper', '~> 1.0'
  s.prepare_command = <<-CMD
    make native
    cp native.a lib/
  CMD
end
